## Getting start

Following steps are required for running wormhole. 

1. Preparing
   - Build source 
   - Run the sample application in local
2. Run server at public cloud
3. Apply a channel through rest-api
4. Register client with the channel id getting from last step
5. Call service deploying at local through cloud

### Preparing

**Build source**

```sh
# Build the wormhole application
$ go build -o wormhole main.go
$ chomod +x wormhole
# Build the mockup web application
$ go build -o fvt/mserver/rest_server fvt/mserver/rest_server.go
```

**Run the sample application in local**

We provide a sample mockup server, which listens at `9081` port and implements several simple rest services.

```sh
$ fvt/mserver/rest_server
```

### Run server at public cloud

You can create a server at AWS, and then run `wormhole ` server application. Also, please create a folder named `etc`, and then copy the configuration `server.yaml` into `etc` folder. 

After it run successfully, it listens QUIC channel at `4242` port, and rest service at `9999` port. Let's suppose the wormhole server is running at `http://manager.emqx.io/`

```shell
$ ./wormhole
```

### Apply a channel through rest-api

From any computer that can access `http://manager.emqx.io/`, and type below command.

```shell
$ curl http://manager.emqx.io:9999/nodes/register -X POST -d '{"name": "node1", "Description": "The demo node."}'
```

It will return result as following. Please notice the field `identifier`, which is the id for the QUIC channel.

```json
{
  "name":"node1",
  "identifier":"04d63e52-4f58-11eb-accc-f45c89b00d3d",
  "description":"The demo node."
}
```

Below step register an endpoint named `kuiper` into the channel (with id `04d63e52-4f58-11eb-accc-f45c89b00d3d`) that created in last command. Also please notice that,

- The `kuiper` endpoint running at 9082 port.
- The root path for `kuiper` endpoint is `/`.

```shell
$ curl http://manager.emqx.io:9999/nodes/04d63e52-4f58-11eb-accc-f45c89b00d3d/mware -X POST -d '{"name": "kuiper", "port": 9082, "path": "/"}'
```

### Register client with the channel id getting from last step

Run following command to register agent. 

- The 1st argument means running `wormhole` with `client` mode.
- The 2nd argument is the id that registered from previous step.

```shell
$ ./wormhole client 04d63e52-4f58-11eb-accc-f45c89b00d3d
```

### Call service deploying at local through cloud

With below command to get the streams defined in local.

```shell
curl http://manager.emqx.io:9999/wh/04d63e52-4f58-11eb-accc-f45c89b00d3d/kuiper/streams
```

Below is the returned result.

```json
["demo1","demo2"]
```



