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
	resp, err := http.Post(fmt.Sprintf("http://%s/candidate", addr), // nolint:noctx
		"application/json; charset=utf-8", bytes.NewReader(payload))
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

func main() { // nolint:gocognit
	offerAddr := flag.String("offer-address", "localhost:50000", "Address that the Offer HTTP server is hosted on.")
	answerAddr := flag.String("answer-address", ":60000", "Address that the Answer HTTP server is hosted on.")
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
		if err := peerConnection.Close(); err != nil {
			fmt.Printf("cannot close peerConnection: %v\n", err)
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
		} else if onICECandidateErr := signalCandidate(*offerAddr, c); onICECandidateErr != nil {
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
	http.HandleFunc("/sdp", func(w http.ResponseWriter, r *http.Request) { // nolint: revive
		sdp := webrtc.SessionDescription{}
		if err := json.NewDecoder(r.Body).Decode(&sdp); err != nil {
			panic(err)
		}

		if err := peerConnection.SetRemoteDescription(sdp); err != nil {
			panic(err)
		}

		// Создание ответа для отправки другому процессу
		answer, err := peerConnection.CreateAnswer(nil)
		if err != nil {
			panic(err)
		}

		// Отправка ответа на HTTP-сервер, прослушивающий другой процесс
		payload, err := json.Marshal(answer)
		if err != nil {
			panic(err)
		}
		resp, err := http.Post(fmt.Sprintf("http://%s/sdp", *offerAddr), "application/json; charset=utf-8", bytes.NewReader(payload)) // nolint:noctx
		if err != nil {
			panic(err)
		} else if closeErr := resp.Body.Close(); closeErr != nil {
			panic(closeErr)
		}

		// Установка LocalDescription и запуск UDP listeners
		err = peerConnection.SetLocalDescription(answer)
		if err != nil {
			panic(err)
		}

		candidatesMux.Lock()
		for _, c := range pendingCandidates {
			onICECandidateErr := signalCandidate(*offerAddr, c)
			if onICECandidateErr != nil {
				panic(onICECandidateErr)
			}
		}
		candidatesMux.Unlock()
	})

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

	// Регистрация обработки создания канала данных
	peerConnection.OnDataChannel(func(d *webrtc.DataChannel) {
		fmt.Printf("New DataChannel %s %d\n", d.Label(), d.ID())

		// Регистрация обработки открытия канала данных
		d.OnOpen(func() {
			fmt.Printf("Data channel '%s'-'%d' open. Random messages will now be sent to any connected DataChannels every 5 seconds\n", d.Label(), d.ID())

			ticker := time.NewTicker(5 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				message, sendTextErr := randutil.GenerateCryptoRandomString(15, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
				if sendTextErr != nil {
					panic(sendTextErr)
				}

				// Отправка сообщения, как текста
				fmt.Printf("Sending '%s'\n", message)
				if sendTextErr = d.SendText(message); sendTextErr != nil {
					panic(sendTextErr)
				}
			}
		})

		// Регистрация обработки текстовых сообщений
		d.OnMessage(func(msg webrtc.DataChannelMessage) {
			fmt.Printf("Message from DataChannel '%s': '%s'\n", d.Label(), string(msg.Data))
		})
	})

	// Запуск HTTP-сервера, который принимает запросы от процесса предложения обмена SDP и ICE кандидатами
	// nolint: gosec
	panic(http.ListenAndServe(*answerAddr, nil))
}
