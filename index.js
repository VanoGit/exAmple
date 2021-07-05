const createButton = document.getElementById('createButton');
createButton.addEventListener('click', createAction);

const callingButton = document.getElementById('callingButton');
callingButton.addEventListener('click', callingAction);

const iceStateButton = document.getElementById('showicestate');
iceStateButton.addEventListener('click', iceStateAction);

let RoomID

function iceStateAction() {
    console.log(peerRef.iceConnectionState);
}

async function createAction() {
    console.log('createAction');
    const resp = await fetch("http://localhost:8000/create");
    const {room_id} = await resp.json();
    RoomID = room_id
    console.log(room_id);
}

// Define peer connections, streams and video elements.
const userVideo = document.getElementById('userVideo');
let userStream;

let peerRef;
let webSocketRef;

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
            `ws://localhost:8000/join?roomID=` + RoomID.toString()
        );

        webSocketRef.addEventListener("open", () => {
            console.log('open')
            callUser();
           // webSocketRef.send(JSON.stringify({join: true}));
        });

        webSocketRef.addEventListener("message", async (e) => {
            console.log('message')
            const message = JSON.parse(e.data);
            console.log(message)
            /*
            if (message.join) {
                callUser();
            }*/

           /* if (message.offer) {
                handleOffer(message.offer);
            }*/

            if (message.answer) {
                console.log("Receiving Answer");
                peerRef.setRemoteDescription(
                    new RTCSessionDescription(message.answer)
                ).then(() => {
                   console.log("setting front answer SUCCESS");
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

    /*
    const handleOffer = async (offer) => {
        console.log("Received Offer, Creating Answer");
        peerRef = createPeer();

        await peerRef.setRemoteDescription(
            new RTCSessionDescription(offer)
        );

        userStream.getTracks().forEach((track) => {
            peerRef.addTrack(track, userStream);
        });

        const answer = await peerRef.createAnswer();
        await peerRef.setLocalDescription(answer);

        webSocketRef.send(
            JSON.stringify({answer: peerRef.localDescription})
        );
    };*/

    const callUser = () => {
        console.log("Calling Other User");
        peerRef = createPeer();
        console.log("START gathering change state " + peerRef.iceGatheringState);


        userStream.getTracks().forEach((track) => {
            console.log("add track " + track.label);
            peerRef.addTrack(track, userStream);
        });
    };

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
            await peerRef.setLocalDescription(myOffer);

            webSocketRef.send(
                JSON.stringify({offer: JSON.stringify(myOffer)})
            );
        } catch (err) {
        }
    };

    const handleIceCandidateEvent = (e) => {
        console.log("Found Ice Candidate");
        if (e.candidate) {
            console.log(e.candidate);
            webSocketRef.send(
                JSON.stringify({iceCandidate: JSON.stringify(e.candidate)})
            );
        }
    };

    const handleiceconnectionstatechange = (e) => {
        console.log("ice connection change state " + JSON.stringify(e));
    };

    const handleicegatheringstatechange = (e) => {
        console.log("ice gathering change state " + peerRef.iceGatheringState);
    };

    const handleTrackEvent = (e) => {
        console.log("Received Tracks");
    };
}