* Just practicing

Reference:

* Travis Jeffery - Distributed Services with Go


Rebuild image container
```shell
make build-docker     
```

Load image container
```shell
kind load docker-image github.com/sharop/calli:0.0.1
```
Installing 
```shell
helm install calli deploy/calli      
```
Uninstalling
```shell
helm uninstall calli    
```
Monitoring
```shell
kubectl logs calli-0  
```
```shell
kubectl get pods --all-namespaces 
```
