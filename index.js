const callingButton = document.getElementById('callingButton');
callingButton.addEventListener('click', callingAction);

// Define peer connections, streams and video elements.
const userVideo = document.getElementById('userVideo');
let userStream;

let peerRef;
let webSocketRef;
let candidates = [];
let completeGathering = false;

function callingAction() {
    console.log('callingAction')
    const openCamera = async () => {
        const allDevices = await navigator.mediaDevices.enumerateDevices();
        const cameras = allDevices.filter(
            (device) => device.kind == "videoinput"
        );
        console.log(cameras);

        const constraints = {
            audio: true,
            video: true,
        };

        try {
            return await navigator.mediaDevices.getUserMedia(constraints);
        } catch (err) {
            console.log(err);
        }
    };


    openCamera().then((stream) => {
        userVideo.srcObject = stream;
        userStream = stream;

        webSocketRef = new WebSocket(
            `ws://localhost:8000/join`
        );

        webSocketRef.addEventListener("open", () => {
            console.log('open')
            console.log("Calling Other User");
            peerRef = createPeer();
            console.log("START gathering change state " + peerRef.iceGatheringState);


            userStream.getTracks().forEach((track) => {
                console.log("add track " + track.label);
                peerRef.addTrack(track, userStream);
            });
        });

        webSocketRef.addEventListener("message", async (e) => {
            console.log('message')
            const message = JSON.parse(e.data);
            console.log(JSON.stringify(message))

            if (message.answer) {
                console.log("Receiving Answer. Start to send candidates");
                peerRef.setRemoteDescription(
                    new RTCSessionDescription(message.answer)
                ).then(() => {
                    console.log("setting front answer SUCCESS");
                    console.log(candidates.length)
                    while(candidates.length == 0) {

                    }
                    console.log("Sending candidates");
                    candidates.forEach(function(candidate, i, arr) {
                        console.log("Sending candidate " + JSON.stringify(candidate));
                        webSocketRef.send(
                            JSON.stringify({iceCandidate: JSON.stringify(candidate)})
                        );
                    });
                }).catch(() => {
                    console.log("setting front answer  ERROR");
                });

            }

            if (message.iceCandidate) {
                console.log("Receiving and Adding ICE Candidate");
                console.log(message)
                try {
                    await peerRef.addIceCandidate(
                        message.iceCandidate
                    ).then(() => {
                        console.log("setting front iceCandidate SUCCESS");
                    }).catch(() => {
                        console.log("setting front iceCandidate ERROR");
                    });
                } catch (err) {
                    console.log("Error Receiving ICE Candidate", err);
                }
            }
        });
    });

    const createPeer = () => {
        console.log("Creating Peer Connection");
        const peer = new RTCPeerConnection({
            iceServers: [{urls: "stun:stun.l.google.com:19302"}],
        });

        peer.onnegotiationneeded = handleNegotiationNeeded;
        peer.onicecandidate = handleIceCandidateEvent;
        peer.oniceconnectionstatechange = handleiceconnectionstatechange;
        peer.onicegatheringstatechange = handleicegatheringstatechange;
        peer.ontrack = handleTrackEvent;

        return peer;
    };

    const handleNegotiationNeeded = async () => {
        console.log("handleNegotiationNeeded, Creating Offer");

        try {
            const myOffer = await peerRef.createOffer();
            peerRef.setLocalDescription(
                myOffer
            ).then(() => {
                console.log("setting front offer SUCCESS");
            }).catch(() => {
                console.log("setting front offer  ERROR");
            });
            webSocketRef.send(
                JSON.stringify({offer: JSON.stringify(myOffer)})
            );

        } catch (err) {
        }
    };

    const handleIceCandidateEvent = (e) => {
        console.log("Found Ice Candidate");
        if (e.candidate) {
            candidates.push(e.candidate);
        }
    };

    const handleiceconnectionstatechange = (e) => {
        console.log("ice connection change state " + JSON.stringify(e));
    };

    const handleicegatheringstatechange = (e) => {
        if (peerRef.iceGatheringState === 'complete') {
            completeGathering = true
        }
        console.log("ice gathering change state " + peerRef.iceGatheringState);
    };

    const handleTrackEvent = (e) => {
        console.log("Received Tracks");
    };
}