package log

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
	api "github.com/sharop/lab_goconnect/api/v1"
	"google.golang.org/protobuf/proto"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"
)
//Rev 1
type DistributedLog struct{
	config 		Config
	log 		*Log
	raft 		*raft.Raft
}

func NewDistributedLog(dataDir string, config Config)(
	*DistributedLog,
	error,
){
	l := &DistributedLog{
		config: config,
	}
	if err := l.setupLog(dataDir); err != nil {
		return nil, err
	}
	if err := l.setupRaft(dataDir); err != nil {
		return nil, err
	}
	return l, nil
}

//Creates the log for this server. The server will store the user's records.
func (l *DistributedLog) setupLog(dataDir string) error {
	logDir := filepath.Join(dataDir, "log")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}
	var err error
	l.log, err = NewLog(logDir, l.config)
	return err
}

// Rev 1
func (l *DistributedLog) setupRaft(dataDir string) error {
	fsm := &fsm{log: l.log}

	logDir := filepath.Join(dataDir, "raft", "log")
	if err := os.MkdirAll(logDir, 0755); err != nil{
		return err
	}
	logConfig := l.config
	logConfig.Segment.InitialOffset = 1
	//Use store for something that I don't understand
	// Configure to initialize in offset 1.
	log.Print("Setup Rafts")
	logStore, err := newLogStore(logDir,logConfig)
	if err != nil{
		return err
	}

	// Create Bolt store required by raft.
	// Key value store where raft stores their metadata.
	stableStore, err := raftboltdb.NewBoltStore(
		filepath.Join(dataDir, "raft", "stable"),
		)
	if err != nil {
		return err
	}

	retain := 1
	// Snapshot to recover and restore data efficiently, when necessary.
	// Rather than send stream of data to leader the new server restore from the sanpshot,
	// and then get the latest changes from the leader.
	snapshotStore, err := raft.NewFileSnapshotStore(
		filepath.Join(dataDir, "raft"),
		retain,
		os.Stderr,
	)
	if err != nil {
		return err
	}

	maxPool := 5
	timeout := 10 * time.Second
	transport := raft.NewNetworkTransport(
		l.config.Raft.StreamLayer,
		maxPool,
		timeout,
		os.Stderr,
	)

	config := raft.DefaultConfig()
	// Unique ID for this server, is the only config field we must set.
	config.LocalID = l.config.Raft.LocalID
	if l.config.Raft.HeartbeatTimeout != 0{
		config.HeartbeatTimeout = l.config.Raft.HeartbeatTimeout
	}
	if l.config.Raft.ElectionTimeout != 0{
		config.ElectionTimeout = l.config.Raft.ElectionTimeout
	}
	if l.config.Raft.LeaderLeaseTimeout != 0 {
		config.LeaderLeaseTimeout = l.config.Raft.LeaderLeaseTimeout
	}
	if l.config.Raft.CommitTimeout != 0 {
		config.CommitTimeout = l.config.Raft.CommitTimeout
	}

	l.raft, err = raft.NewRaft(
		config,
		fsm,
		logStore,
		stableStore,
		snapshotStore,
		transport,
	)

	if err != nil {
		return err
	}

	hasState, err := raft.HasExistingState(
		logStore,
		stableStore,
		snapshotStore,
	)
	if err != nil {
		return err
	}

	if l.config.Raft.Bootstrap && !hasState {
		config := raft.Configuration{
			Servers: []raft.Server{{
				ID:      config.LocalID,
				Address: raft.ServerAddress(l.config.Raft.BindAddr), //transport.LocalAddr(),
			}},
		}
		err = l.raft.BootstrapCluster(config).Error()
	}
	return err
}

//API that append records to and read records from the log and wrap Raft.
// The distributed log will have the same API as the Log type.
// Append the records to the log. Unlike append the record directly.
func (l *DistributedLog) Append(record *api.Record) (uint64 , error ){
	res, err := l.apply(
		AppendRequestType,
		&api.ProduceRequest{Record: record},
	)
	if err!=nil{
		return 0, err
	}
	return res.(*api.ProduceResponse).Offset, nil
}

// Rev 1
func (l *DistributedLog) apply(reqType RequestType, req proto.Message)(
	interface{},
	error,
	) {
	var buf bytes.Buffer
	_, err := buf.Write([]byte{byte(reqType)})
	if err != nil {
		return nil, err
	}
	b, err := proto.Marshal(req)
	if err != nil {
		return nil, err
	}
	_, err = buf.Write(b)
	if err != nil {
		return nil, err
	}
	timeout := 10 * time.Second
	future := l.raft.Apply(buf.Bytes(), timeout)
	if future.Error() != nil {
		return nil, future.Error()
	}
	res := future.Response()
	if err, ok := res.(error); ok {
		return nil, err
	}
	return res, nil
}

// relaxed consistency, this not go through raft.
func (l *DistributedLog) Read(offset uint64) (*api.Record, error){
	return l.log.Read(offset)
}

/*
Discovery Integration.
Integration of Raft to discovery service. (Serf-driven discovery)
*/

// Adds the server to the Raft cluster.
func (l *DistributedLog) Join(id, addr string) error {
	configFuture := l.raft.GetConfiguration()
	if err := configFuture.Error(); err != nil {
		return err
	}
	serverID := raft.ServerID(id)
	serverAddr := raft.ServerAddress(addr)
	for _, srv := range configFuture.Configuration().Servers {
		if srv.ID == serverID || srv.Address == serverAddr {
			if srv.ID == serverID && srv.Address == serverAddr {
				// we already know this server.
				return nil
			}
			removeFuture := l.raft.RemoveServer(serverID, 0, 0)
			if err := removeFuture.Error(); err != nil {
				return err
			}
		}
	}
	//Add each server as voter.
	addFuture := l.raft.AddVoter(serverID, serverAddr, 0, 0)
	if err := addFuture.Error(); err != nil {
		return err
	}
	return nil
}

func (l *DistributedLog) Leave(id string) error {
	removeFuture := l.raft.RemoveServer(raft.ServerID(id), 0, 0)
	return removeFuture.Error()
}

//Blocks until the cluster has elected aleader or times out.
func (l *DistributedLog) WaitForLeader(timeout time.Duration) error {
	timeoutc := time.After(timeout)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-timeoutc:
			return fmt.Errorf("timed out")
		case <-ticker.C:
			if l := l.raft.Leader(); l != "" {
				return nil
			}
		}
	}
}

func (l *DistributedLog) Close() error{
	f := l.raft.Shutdown()
	if err := f.Error(); err != nil {
		return err
	}
	return l.log.Close()
}

//Exposes raft server data.
func (l *DistributedLog) GetServers()([]*api.Server, error){
	future := l.raft.GetConfiguration()
	if err:= future.Error(); err != nil{
		return nil, err
	}
	var servers []*api.Server
	for _, server := range future.Configuration().Servers{
		servers = append(servers, &api.Server{
			Id:				string(server.ID),
			RpcAddr:		string(server.Address),
			IsLeader: 		l.raft.Leader() == server.Address,
		})
	}
	return servers, nil
}

/// Mas arriba el error.

// FSM
/*
The FSM must access the data it manages.
The FSM append records to the log.
*/


var _ raft.FSM = (*fsm)(nil)

type fsm struct {
	log *Log
}


type RequestType uint8

const (
	AppendRequestType RequestType = 0
)

// Raft invokes after commiting a log entry.
func (l *fsm) Apply(record *raft.Log) interface{} {
	buf := record.Data
	reqType := RequestType(buf[0])
	switch reqType {
	case AppendRequestType:
		return l.applyAppend(buf[1:])
	}
	return nil
}

func (l *fsm) applyAppend(b []byte) interface{} {
	var req api.ProduceRequest
	err := proto.Unmarshal(b, &req)
	if err != nil {
		return err
	}
	offset, err := l.log.Append(req.Record)
	if err != nil {
		return err
	}
	return &api.ProduceResponse{Offset: offset}
}

// Create a snapshot periodically, avoid to send all the log through raft.
// Point in time snapshot of the FSM
func (f *fsm) Snapshot() (raft.FSMSnapshot, error) {
	r := f.log.Reader()
	return &snapshot{reader: r}, nil
}

var _ raft.FSMSnapshot = (*snapshot)(nil)
type snapshot struct{
	reader io.Reader
}

func (s *snapshot) Persist(sink raft.SnapshotSink) error {
	if _,err := io.Copy(sink, s.reader); err != nil{
		_=sink.Cancel()
		return err
	}
	return sink.Close()
}

func (s *snapshot) Release() {}

// Restore FSM from Snapshot.
func (f *fsm) Restore(closer io.ReadCloser) error {
	b := make([]byte, lenWidth)
	var buf bytes.Buffer
	for i := 0; ; i++ {
		_, err := io.ReadFull(closer, b)
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		size := int64(enc.Uint64(b))
		if _, err = io.CopyN(&buf, closer, size); err != nil {
			return err
		}
		record := &api.Record{}
		if err = proto.Unmarshal(buf.Bytes(), record); err != nil {
			return err
		}
		if i == 0 {
			f.log.Config.Segment.InitialOffset = record.Offset
			if err := f.log.Reset(); err != nil {
				return err
			}
		}
		if _, err = f.log.Append(record); err != nil {
			return err
		}
		buf.Reset()
	}
	return nil
}
///Arriba esta el error.
/// RAFT log store.
var _ raft.LogStore = (*logStore)(nil)
type logStore struct{
	*Log
}

//Creates new log store for the FSM
func newLogStore(dir string, config Config) (*logStore, error) {
	log, err := NewLog(dir, config)
	if err!= nil{
		return nil, err
	}
	return &logStore{log}, nil
}
// Below methods are API for raft, in order to wrap the log that RAFT requires.
// Raft call indexes and we call offsets
func (l *logStore) FirstIndex() (uint64, error) {
	return l.LowestOffset()
}

func (l *logStore) LastIndex() (uint64, error) {
	off, err := l.HighestOffset()
	return off, err
}

func (l *logStore) GetLog(index uint64, out *raft.Log) error {
	in, err := l.Read(index)
	if err != nil {
		return err
	}
	out.Data = in.Value
	out.Index = in.Offset
	out.Type = raft.LogType(in.Type)
	out.Term = in.Term
	return nil
}


func (l *logStore) StoreLog(record *raft.Log) error {
	return l.StoreLogs([]*raft.Log{record})
}
func (l *logStore) StoreLogs(records []*raft.Log) error {
	for _, record := range records {
		if _, err := l.Append(&api.Record{
			Value: record.Data,
			Term:  record.Term,
			Type:  uint32(record.Type),
		}); err != nil {
			return err
		}
	}
	return nil
}

func (l *logStore) DeleteRange(min, max uint64) error {
	return l.Truncate(max)
}

/** Stream Layer
Provide a low-level  stream abstraction to connect with Raft servers.
**/
var _ raft.StreamLayer = (*StreamLayer)(nil)

type StreamLayer struct{
	ln					net.Listener
	serverTLSConfig		*tls.Config
	peerTLSConfig		*tls.Config
}

func NewStreamLayer(
	ln net.Listener,
	serverTLSConfig,
	peerTLSConfig *tls.Config,
) *StreamLayer {
	return &StreamLayer{
		ln:              ln,
		serverTLSConfig: serverTLSConfig,
		peerTLSConfig:   peerTLSConfig,
	}
}

const RaftRPC = 1
func (s *StreamLayer) Dial(address raft.ServerAddress, timeout time.Duration) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: timeout}
	var conn, err = dialer.Dial("tcp", string(address))
	if err != nil {
		return nil, err
	}
	// identify to mux this is a raft rpc
	_, err = conn.Write([]byte{byte(RaftRPC)})
	if err != nil {
		return nil, err
	}
	if s.peerTLSConfig != nil {
		conn = tls.Client(conn, s.peerTLSConfig)
	}
	return conn, err
}
func (s *StreamLayer) Accept() (net.Conn, error) {
	conn, err := s.ln.Accept()
	if err != nil {
		return nil, err
	}
	b := make([]byte, 1)
	_, err = conn.Read(b)
	if err != nil {
		return nil, err
	}
	if bytes.Compare([]byte{byte(RaftRPC)}, b) != 0 {
		return nil, fmt.Errorf("not a raft rpc")
	}
	if s.serverTLSConfig != nil {
		return tls.Server(conn, s.serverTLSConfig), nil
	}
	return conn, nil
}

func (s *StreamLayer) Close() error {
	return s.ln.Close()
}

func (s *StreamLayer) Addr() net.Addr {
	return s.ln.Addr()
}






