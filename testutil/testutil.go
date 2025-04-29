package testutil

import (
	"net"
	"strconv"
)

func GetAvailablePort() (string, error) {
	a, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return "", err
	}
	l, err := net.ListenTCP("tcp", a)
	if err != nil {
		return "", err
	}
	defer l.Close()
	return strconv.Itoa(l.Addr().(*net.TCPAddr).Port), nil
}

func CombineErrChan(c1, c2 <-chan error) <-chan error {
	errChan := make(chan error)

	go func() {
		for {
			select {
			case err, ok := <-c1:
				errChan <- err
				if !ok {
					c1 = nil
				}
			case err, ok := <-c2:
				errChan <- err
				if !ok {
					c2 = nil
				}
			}

			if c1 == nil && c2 == nil {
				break
			}
		}
	}()

	return errChan
}
