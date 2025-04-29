package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/FluffyKebab/pearly/kademila"
	"github.com/FluffyKebab/pearly/node/basic"
	"github.com/FluffyKebab/pearly/peer"
	"github.com/FluffyKebab/pearly/transport/encrypted"
	"github.com/FluffyKebab/pearly/transport/tcp"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	port, bootstrapAddr := getUserInput()

	// First we initialize the transport the node is going to use to communicate
	// with other nodes. In this case we are using encrypted tcp. In addition to
	// obfuscating the data, this transport also gives our node an ID by
	// hasing the public key the transport uses for encryption. This ID is both
	// used by the dht and in the transport to validate that we are communicating
	// with the correct node. When the transport upgrades the connection, it
	// receives both the ID and the public key of the peer. It then validates
	// that the ID is the hash of the public key. If not the connection is
	// discarded.
	transport, err := encrypted.NewTransport(tcp.New(port))
	if err != nil {
		return err
	}

	// Next we initialize and run a node that uses the transport we just created.
	// We receive all errors that result from handling remote connections in the
	// error chan. The node is also responsible for protocol muxing. E.G.
	// making sure that the right protocol handler is called when we receive
	// a new connection.
	node := basic.New(transport, transport.ID())
	errChan, err := node.Run(context.Background())
	if err != nil {
		return fmt.Errorf("running node: %w", err)
	}
	go func() {
		for {
			err := <-errChan
			fmt.Println(err)
		}
	}()

	// Using the node we created, we can initialize a kadmila dht.
	dht := kademila.New(node)
	if bootstrapAddr != "" {
		// The bootstrap process is how we join an already created network. This
		// implementation is simple, and consists of only two steps: first we
		// try to find a key with the same value as our own node. If there is
		// space in the bootstrap nodes peer store, it will add us as a peer.
		// Lastly we add the node to our own peerstore.
		err = dht.Bootstrap(context.Background(), peer.New(nil, bootstrapAddr))
		if err != nil {
			return fmt.Errorf("bootsrap: %w", err)
		}
	}

	for {
		if err := doCommand(dht); err != nil {
			fmt.Println(err.Error())
		}
	}
}

func doCommand(dht kademila.DHT) error {
	fmt.Println("(add/get) (value/key)")
	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')

	textSplit := strings.Split(text, " ")
	if len(textSplit) < 2 {
		return fmt.Errorf("missing command and/or value")
	}
	command := textSplit[0]
	value := strings.Join(textSplit[1:], " ")

	if command == "add" {
		key, err := hashData([]byte(value))
		if err != nil {
			return fmt.Errorf("hashing value failed: %w", err)
		}

		err = dht.SetValue(context.Background(), key, []byte(value))
		if err != nil {
			return fmt.Errorf("setting value failed: %w", err)
		}

		fmt.Println("stored value in key:", hex.EncodeToString(key))
		return nil
	}
	if command == "get" {
		key, err := hex.DecodeString(strings.TrimSpace(value))
		if err != nil {
			return fmt.Errorf("invalid key: %w", err)
		}

		valueFromDht, err := dht.GetValue(context.Background(), key)
		if err != nil {
			return fmt.Errorf("finding value in dht failed: %w", err)
		}

		fmt.Println("result:", string(valueFromDht))
		return nil
	}

	return fmt.Errorf("unavilable command: %v", command)
}

func getUserInput() (string, string) {
	var port string
	fmt.Println("port:")
	fmt.Scanln(&port)

	var bootsrapAddr string
	fmt.Println("bootsrap addr (optional):")
	fmt.Scanln(&bootsrapAddr)

	return port, bootsrapAddr
}

func hashData(value []byte) ([]byte, error) {
	hash := sha256.New()
	_, err := hash.Write(value)
	return hash.Sum(nil), err
}
