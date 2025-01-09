package kadmilla

import "github.com/FluffyKebab/pearly/node"

/*
protocols:
	- kdmfindkey
		- send: key to be found, k
		- recive: k closest nodes or final value
	- kdmstore
		- send key, value to store in node
	- kdmfindpeer
		- send: key to be found, k
		- recive: k closest nodes or final value

*/

type DHT struct {
	n node.Node
}
