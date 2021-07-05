package server

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
	"log"
	"net/http"
)

// AllRooms is the global hashmap for the server
var AllRooms RoomMap
var webSockets = make([]*websocket.Conn, 0)

// CreateRoomRequestHandler Create a Room and return roomID
func CreateRoomRequestHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	roomID := AllRooms.CreateRoom()

	type resp struct {
		RoomID string `json:"room_id"`
	}

	log.Println(AllRooms.Map)
	json.NewEncoder(w).Encode(resp{RoomID: roomID})
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type broadcastMsg struct {
	Message map[string]interface{}
	RoomID  string
	Client  *websocket.Conn
}

var broadcast = make(chan broadcastMsg)

func broadcaster(peerConnection *webrtc.PeerConnection) {
	for {
		msg := <-broadcast

		fmt.Println("New MSG \n", msg.Message)

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
			response.RoomID = msg.RoomID
			response.Client = msg.Client
			response.Message = map[string]interface{}{"answer":answer}
			err = msg.Client.WriteJSON(response.Message)
			if err != nil {
				fmt.Println("error SENDING", err)
				msg.Client.Close()
			}
		}

		if candidate, ok := msg.Message["iceCandidate"]; ok {

			myCandidate := webrtc.ICECandidateInit{}
			MyDecodeCandidate([]byte(candidate.(string)), &myCandidate)

			err := peerConnection.AddICECandidate(myCandidate)
			if err != nil {
				fmt.Println("error AddICECandidate", err)
			}

		}

	}
}

// JoinRoomRequestHandler will join the client in a particular room
func JoinRoomRequestHandler(w http.ResponseWriter, r *http.Request) {
	roomID, ok := r.URL.Query()["roomID"]

	if !ok {
		log.Println("roomID missing in URL Parameters")
		return
	}
	fmt.Println("Join to the room ", roomID)

	peerConnectionConfig := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302", "stun:stun1.l.google.com:19302"},
			},
		},
	}
	fmt.Println(peerConnectionConfig)
	// Create a new RTCPeerConnection
	peerConnection, err := webrtc.NewPeerConnection(peerConnectionConfig)
	if err != nil {
		fmt.Println("ERROR ", err)
	}

	peerConnection.OnICECandidate(func(c *webrtc.ICECandidate) {
		fmt.Println("BACKEND ICE CANDIDATE ", c)
		if c == nil {
			return
		}
		for _, webSocket := range webSockets {
			msg := broadcastMsg{
				Message: map[string]interface{}{"iceCandidate": c},
				Client: webSocket,
			}

			err = msg.Client.WriteJSON(msg.Message)
			if err != nil {
				fmt.Println("error SENDING", err)
				msg.Client.Close()
			}
		}
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
	})

	peerConnection.OnTrack(func(remoteTrack *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		fmt.Println("ON TRACK!!!!")
	})

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal("Web Socket Upgrade Error", err)
	}
	webSockets = append(webSockets, ws)

	AllRooms.InsertIntoRoom(roomID[0], false, ws)

	go broadcaster(peerConnection)

	for {
		var msg broadcastMsg

		err := ws.ReadJSON(&msg.Message)
		if err != nil {
			log.Fatal("Read Error: ", err)
		}

		msg.Client = ws
		msg.RoomID = roomID[0]

		log.Println(msg.Message)

		broadcast <- msg
	}
}
