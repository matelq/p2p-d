// пример взаимодействия двух экземпляров Pion
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/pion/randutil"
	"github.com/pion/webrtc/v4"
)

func signalCandidate(addr string, c *webrtc.ICECandidate) error {
	payload := []byte(c.ToJSON().Candidate)
	resp, err := http.Post(fmt.Sprintf("http://%s/candidate", addr), "application/json; charset=utf-8", bytes.NewReader(payload)) //nolint:noctx
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

func main() { //nolint:gocognit
	offerAddr := flag.String("offer-address", ":50000", "Address that the Offer HTTP server is hosted on.")
	answerAddr := flag.String("answer-address", "127.0.0.1:60000", "Address that the Answer HTTP server is hosted on.")
	flag.Parse()

	var candidatesMux sync.Mutex
	pendingCandidates := make([]*webrtc.ICECandidate, 0)

	// Подготовка конфигурации
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	// Создание нового RTCPeerConnection
	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		panic(err)
	}
	defer func() {
		if cErr := peerConnection.Close(); cErr != nil {
			fmt.Printf("cannot close peerConnection: %v\n", cErr)
		}
	}()

	// Когда ICE кандидат становится доступен, он отправляется другому экземпляру Pion,
	// который добавляет этого кандидата при помощи вызова AddICECandidate
	peerConnection.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}

		candidatesMux.Lock()
		defer candidatesMux.Unlock()

		desc := peerConnection.RemoteDescription()
		if desc == nil {
			pendingCandidates = append(pendingCandidates, c)
		} else if onICECandidateErr := signalCandidate(*answerAddr, c); onICECandidateErr != nil {
			panic(onICECandidateErr)
		}
	})

	// HTTP-обработчик, который позволяет другому экземпляру Pion отправлять нам ICE кандидатов
	// Это позволяет нам добавлять ICE кандидатов быстрее,
	// так как не нужно ждать кандидатов STUN или TURN, которые могут быть медленнее
	http.HandleFunc("/candidate", func(w http.ResponseWriter, r *http.Request) { //nolint: revive
		candidate, candidateErr := io.ReadAll(r.Body)
		if candidateErr != nil {
			panic(candidateErr)
		}
		if candidateErr := peerConnection.AddICECandidate(webrtc.ICECandidateInit{Candidate: string(candidate)}); candidateErr != nil {
			panic(candidateErr)
		}
	})

	// HTTP-обработчик, который обрабатывает SessionDescription,
	// предоставленный нам другим процессом Pion
	http.HandleFunc("/sdp", func(w http.ResponseWriter, r *http.Request) { //nolint: revive
		sdp := webrtc.SessionDescription{}
		if sdpErr := json.NewDecoder(r.Body).Decode(&sdp); sdpErr != nil {
			panic(sdpErr)
		}

		if sdpErr := peerConnection.SetRemoteDescription(sdp); sdpErr != nil {
			panic(sdpErr)
		}

		candidatesMux.Lock()
		defer candidatesMux.Unlock()

		for _, c := range pendingCandidates {
			if onICECandidateErr := signalCandidate(*answerAddr, c); onICECandidateErr != nil {
				panic(onICECandidateErr)
			}
		}
	})
	// Запуск HTTP-сервера, который принимает запросы от другого процесса
	// nolint: gosec
	go func() { panic(http.ListenAndServe(*offerAddr, nil)) }()

	// Создание канала данных с меткой "данные"
	dataChannel, err := peerConnection.CreateDataChannel("data", nil)
	if err != nil {
		panic(err)
	}

	// Установка обработчика состояния соединения с одноранговым узлом
	// Уведомление о подключении/отключении однорангового узла
	peerConnection.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		fmt.Printf("Peer Connection State has changed: %s\n", s.String())

		if s == webrtc.PeerConnectionStateFailed {
			// Ожидание, пока PeerConnection не будет иметь сетевой активности в течение 30 секунд или произойдет другой сбой
			// В таком случае PeerConnection может быть повторно подключен с помощью перезапуска ICE
			// webrtc.PeerConnectionStateDisconnected используется, если необходимо обнаружить более быстрый тайм-аут
			// PeerConnection может вернуться из PeerConnectionStateDisconnected
			fmt.Println("Peer Connection has gone to failed exiting")
			os.Exit(0)
		}

		if s == webrtc.PeerConnectionStateClosed {
			// PeerConnection был явно закрыт
			// Обычно это происходит из DTLS CloseNotify
			fmt.Println("Peer Connection has gone to closed exiting")
			os.Exit(0)
		}
	})

	// Регистрация обработки открытия канала
	dataChannel.OnOpen(func() {
		fmt.Printf("Data channel '%s'-'%d' open. Random messages will now be sent to any connected DataChannels every 5 seconds\n", dataChannel.Label(), dataChannel.ID())

		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			message, sendTextErr := randutil.GenerateCryptoRandomString(15, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
			if sendTextErr != nil {
				panic(sendTextErr)
			}

			// Отправка сообщения, как текста
			fmt.Printf("Sending '%s'\n", message)
			if sendTextErr = dataChannel.SendText(message); sendTextErr != nil {
				panic(sendTextErr)
			}
		}
	})

	// Регистрация обработки текстовых сообщений
	dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
		fmt.Printf("Message from DataChannel '%s': '%s'\n", dataChannel.Label(), string(msg.Data))
	})

	// Создание предложения для отправки в другой процесс
	offer, err := peerConnection.CreateOffer(nil)
	if err != nil {
		panic(err)
	}

	// Установка LocalDescription и запуск UDP listeners
	// Это запустит сбор кандидатов ICE
	if err = peerConnection.SetLocalDescription(offer); err != nil {
		panic(err)
	}

	// Отправка предложения на HTTP-сервер другого процесса
	payload, err := json.Marshal(offer)
	if err != nil {
		panic(err)
	}
	resp, err := http.Post(fmt.Sprintf("http://%s/sdp", *answerAddr), "application/json; charset=utf-8", bytes.NewReader(payload)) // nolint:noctx
	if err != nil {
		panic(err)
	} else if err := resp.Body.Close(); err != nil {
		panic(err)
	}

	// Вечная блокировка
	select {}
}
