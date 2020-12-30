package amcp_test

import (
	"log"

	"github.com/sean-callahan/amcp"
)

func Example() {
	// Connect to the remote AMCP server.
	c, err := amcp.Dial("127.0.0.1:5250")
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	// Send the PLAY command to make channel 1 white.
	_, _, err = c.Do("PLAY", 1, "#FFFFFF")
	if err != nil {
		log.Fatal(err)
	}
}
