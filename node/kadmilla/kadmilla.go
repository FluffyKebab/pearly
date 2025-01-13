package kadmilla

/*
protocols:
	- kdmfindkey
		- send: key to be found, k
		- recive: k closest nodes or final value
	- kdmstore
		- send key, value to store in node
	- kdmfindpeer
		- send: key to be found, k
		- recive: k closest nodes

RegisterFindKey(n node.Node, peerStore peer.PeerStore) FindKeyService
*/
