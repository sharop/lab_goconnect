package main

import (
	"context"
	"flag"
	"fmt"
	api "github.com/sharop/lab_goconnect/api/v1"
	"google.golang.org/grpc"
	"log"
)

func main(){
	addr := flag.String("addr", ":10500", "service address")
	flag.Parse()
	conn, err := grpc.Dial(*addr, grpc.WithInsecure())
	if err != nil{
		log.Fatalln(err)
	}
	client := api.NewLogClient(conn)
	ctx := context.Background()
	res, err := client.GetServers(ctx, &api.GetServersRequest{})
	if err != nil{
		log.Fatal(err)
	}
	fmt.Println("servers:")
	for _, server := range res.Servers{
		fmt.Printf("\t- %v\n", server)
	}
}
