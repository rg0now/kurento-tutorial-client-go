# Kurento tutorial client in Go 

This quick hack implements the client side of the [Kurento](https://www.kurento.org/) [one-to-one
video call
tutorial](https://doc-kurento.readthedocs.io/en/latest/tutorials/node/tutorial-one2one.html). It
registers with the application server,  and implements the caller and callee state machines to
initiate a connection, exchange answer/offer, wait for ICE to connect, and then send a predefined
video over from the caller to the callee which will then save the received video to a file.

This client is used to run demos for the [STUNner Kubernetes ingress gateway for
WebRTC](https://github.com/l7mp/stunner).

## Getting started

## Install
The usual:
``` console
cd kurento-tutorial-client-go/
go build ./...
```

## Prepare video
### H264
Must use the NAL frame format for streaming. Recode video:
``` console
ffmpeg -i sample_640x360.mkv -an -vcodec libx264 sample_640x360.mkv
```

### VP8
Must use the IVF media container for streaming. Recode video:
``` console
ffmpeg -i sample_640x360.mkv  -vcodec libvpx -s 640x360 sample_640x360.ivf
```

## Configure
The code assumes the TURN server runs at the default port UDP/3478 and uses `plaintext`
authentication with `user/pass`. (If not, rewrite the `webrtc.Configuration` struct in the source).
Then, identify the public IP address of the TURN server, e.g., for STUNner:
``` console
$ export TURN_SERVER_ADDR=$(kubectl get svc stunner -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
```
We also need the public facing address of the Application server to create, manage and tear-down
WebRTC sessions via the WebSocket control connection. For the STUNner tutorials:
``` console
$ export APPLICATION_SERVER_ADDR=$(kubectl get svc webrtc-server -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
$ export APPLICATION_SERVER_PORT=$(kubectl get svc webrtc-server -o jsonpath='{.spec.ports[0].port}')
```

## Start client
### Without transcoding
Send/receive the same encoding:
* Sender side:
``` console
go run webrtc-client.go caller --peer=test2 --ice-addr="${TURN_SERVER_ADDR}" --url="wss://${APPLICATION_SERVER_ADDR}:${APPLICATION_SERVER_ADDR}/one2one" --debug -file=sample/sample_640x360.ivf
```
* Receiver side: 
``` console
go run webrtc-client.go callee --user=test2 --ice-addr="${TURN_SERVER_ADDR}" --url="wss://${APPLICATION_SERVER_ADDR}:${APPLICATION_SERVER_ADDR}/one2one" --debug -file=/tmp/output.ivf
```
### With transcoding
Send H264, receive VP8:
* Sender side:
``` console
go run webrtc-client.go caller --peer=test2 --ice-addr="${TURN_SERVER_ADDR}" --url="wss://${APPLICATION_SERVER_ADDR}:${APPLICATION_SERVER_ADDR}/one2one" --debug -file=sample/sample_640x360.h264
```
* Receiver side: 
``` console
go run webrtc-client.go callee --user=test2 --ice-addr="${TURN_SERVER_ADDR}" --url="wss://${APPLICATION_SERVER_ADDR}:${APPLICATION_SERVER_ADDR}/one2one" --debug -file=/tmp/output.ivf
```

## Start magic-mirror background traffic
Create an ivf or h264 file to be played and run the below script.
```console
demo/run-mirror-traffic.sh -n <NUMBER_OF_CALLS> -m <MODE[rolling|static]> -f <FILE-TO-PLAY>
```

## Help

STUNner development is coordinated on Discord, send [us](https://github.com/l7mp/stunner/blob/main/AUTHORS) an email to ask an invitation.

## License

Copyright 2021-2022 by its authors. Some rights reserved. See [AUTHORS](https://github.com/l7mp/stunner/blob/main/AUTHORS).

MIT License - see [LICENSE](/LICENSE) for full text.

## Acknowledgments

Demo adopted from [Kurento](https://www.kurento.org). Initial code adopted from
[pion/webrtc](https://github.com/pion/webrtc) examples.

