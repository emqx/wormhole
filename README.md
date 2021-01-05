# wormhole
The channel for routing request from cloud to edge.



http://127.0.0.1:9081/stream

client1 agent <----|> server (4242 - Quic server)    //, http://em.emqx.io:9082 - Rest server)
client2 agent <----|
               QUIC

  1) User GET: http://em.emqx.io:9082/client1/stream
  2) Encode request, and send it to agent
  3) Agent send request to real target - http://192.168.10.2:9081/stream
  4) Agent send result to server
  5) Server send result to user
