package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
	"log"
	"net/http"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type broadcastMsg struct {
	Message map[string]interface{}
}

var peerConnection *webrtc.PeerConnection
var candidates = make([]*webrtc.ICECandidate, 0)
var webSockets = make([]*websocket.Conn, 0)
var completeGathering bool
var isFirstCandidate = true

// JoinRoomRequestHandler will join the client in a particular room
func sendCandidatesFromServer() {
	for _, webSocket := range webSockets {

		for _, candidate := range candidates {
			msg := broadcastMsg{
				Message: map[string]interface{}{"iceCandidate": candidate},
			}

			err := webSocket.WriteJSON(msg.Message)
			if err != nil {
				fmt.Println("error SENDING", err)
				webSocket.Close()
			}
		}

	}
}

func JoinRoomRequestHandler(w http.ResponseWriter, r *http.Request) {
	peerConnectionConfig := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302", "stun:stun1.l.google.com:19302"},
			},
		},
	}

	// Create a new RTCPeerConnection
	var err error
	peerConnection, err = webrtc.NewPeerConnection(peerConnectionConfig)
	if err != nil {
		fmt.Println("ERROR ", err)
	}

	peerConnection.OnICECandidate(func(c *webrtc.ICECandidate) {
		fmt.Println("BACKEND ICE CANDIDATE ", c)
		if c == nil {
			return
		}
		candidates = append(candidates, c)
	})

	peerConnection.OnICEConnectionStateChange(func(cs webrtc.ICEConnectionState) {
		log.Println("Ice connection changed to ", cs.String())
		if cs == webrtc.ICEConnectionStateFailed {
			log.Println("Closing peer connection as ICE connection failed")
			peerConnection.Close()
		}
	})

	peerConnection.OnICEGatheringStateChange(func(gs webrtc.ICEGathererState) {
		log.Println("Gathering changed to ", gs.String())
		if gs.String() == "complete" {
			completeGathering = true
		}
	})

	peerConnection.OnTrack(func(remoteTrack *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		fmt.Println("ON TRACK!!!!")
	})

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal("Web Socket Upgrade Error", err)
	}
	webSockets = append(webSockets, ws)

	for {
		var msg broadcastMsg

		err := ws.ReadJSON(&msg.Message)
		if err != nil {
			log.Fatal("Read Error: ", err)
		}

		if offer, ok := msg.Message["offer"]; ok {

			myOffer := webrtc.SessionDescription{}
			MyDecodeSDP([]byte(offer.(string)), &myOffer)
			err := peerConnection.SetRemoteDescription(myOffer)
			if err != nil {
				fmt.Println("ERROR ", err)
			}

			answer, err := peerConnection.CreateAnswer(nil)
			if err != nil {
				fmt.Println("error", err)
			}
			err = peerConnection.SetLocalDescription(answer)
			if err != nil {
				fmt.Println("error", err)
			}

			var response broadcastMsg
			response.Message = map[string]interface{}{"answer": answer}
			err = ws.WriteJSON(response.Message)
			if err != nil {
				fmt.Println("error SENDING", err)
				ws.Close()
			}
		}

		if candidate, ok := msg.Message["iceCandidate"]; ok {

			myCandidate := webrtc.ICECandidateInit{}
			MyDecodeCandidate([]byte(candidate.(string)), &myCandidate)

			err := peerConnection.AddICECandidate(myCandidate)
			if err != nil {
				fmt.Println("error AddICECandidate", err)
			}

			if isFirstCandidate {
				sendCandidatesFromServer()
				isFirstCandidate = false
			}

		}
	}
}

func main() {
	http.HandleFunc("/join", JoinRoomRequestHandler)

	log.Println("Starting Server on Port 8000")
	err := http.ListenAndServe(":8000", nil)
	if err != nil {
		log.Fatal(err)
	}
}
