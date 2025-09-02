package network

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"
)

const DefaultPort = 42567

type PeerNetwork struct {
	peers     []string
	listener  net.Listener
	mu        sync.RWMutex
	callbacks map[string]func([]byte)
}

type Message struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

func NewPeerNetwork() *PeerNetwork {
	return &PeerNetwork{
		peers:     make([]string, 0),
		callbacks: make(map[string]func([]byte)),
	}
}

func (pn *PeerNetwork) AddPeer(address string) {
	pn.mu.Lock()
	defer pn.mu.Unlock()
	pn.peers = append(pn.peers, address)
}

func (pn *PeerNetwork) RegisterCallback(msgType string, callback func([]byte)) {
	pn.mu.Lock()
	defer pn.mu.Unlock()
	pn.callbacks[msgType] = callback
}

func (pn *PeerNetwork) Start(port int) error {
	var err error
	pn.listener, err = net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}

	go pn.acceptConnections()
	return nil
}

func (pn *PeerNetwork) acceptConnections() {
	for {
		conn, err := pn.listener.Accept()
		if err != nil {
			continue
		}
		go pn.handleConnection(conn)
	}
}

func (pn *PeerNetwork) handleConnection(conn net.Conn) {
	defer conn.Close()

	decoder := json.NewDecoder(conn)
	var msg Message

	if err := decoder.Decode(&msg); err != nil {
		return
	}

	pn.mu.RLock()
	callback, exists := pn.callbacks[msg.Type]
	pn.mu.RUnlock()

	if exists {
		data, _ := json.Marshal(msg.Data)
		callback(data)
	}
}

func (pn *PeerNetwork) Broadcast(msgType string, data any) {
	pn.mu.RLock()
	peers := make([]string, len(pn.peers))
	copy(peers, pn.peers)
	pn.mu.RUnlock()

	msg := Message{Type: msgType, Data: data}
	msgData, _ := json.Marshal(msg)

	for _, peer := range peers {
		go func(addr string) {
			conn, err := net.Dial("tcp", addr)
			if err != nil {
				return
			}
			defer conn.Close()
			conn.Write(msgData)
		}(peer)
	}
}
