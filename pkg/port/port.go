package port

import (
	"fmt"
	"net"
	"net/http"
	"time"
)

// GetFreePort returns a free port number. If the default port
// is not available, it will try to find a free port, checking
// at most the next 1024 ports. If no free port is found, it
// will get a random free port. If that fails, it will return
// the default port. If the default port is set to 0, 8080 will
// be used instead.
func GetFreePort(defaultPort int) (int, error) {
	if defaultPort == 0 {
		defaultPort = 8080
	}

	if CheckAvailability(defaultPort) {
		return defaultPort, nil
	}

	for i := range 1024 {
		port := defaultPort + i
		if port >= 0 && port <= 65535 {
			if CheckAvailability(port) {
				return port, nil
			}
		} else {
			break
		}
	}

	port, err := GetRandomFreePort()
	if err != nil {
		return defaultPort, err
	}

	return port, nil
}

func CheckAvailability(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	defer ln.Close()

	return !LocalhostReturnsOK(port, 30*time.Millisecond)
}

func LocalhostReturnsOK(port int, timeout time.Duration) bool {
	client := &http.Client{Timeout: timeout}
	resp, err := client.Get(fmt.Sprintf("http://localhost:%d", port))
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

func GetRandomFreePort() (port int, err error) {
	// Asks the kernel for a free open port that is ready to use.
	// Credit: https://gist.github.com/sevkin/96bdae9274465b2d09191384f86ef39d
	var a *net.TCPAddr
	if a, err = net.ResolveTCPAddr("tcp", "localhost:0"); err == nil {
		var l *net.TCPListener
		if l, err = net.ListenTCP("tcp", a); err == nil {
			defer l.Close()
			return l.Addr().(*net.TCPAddr).Port, nil
		}
	}
	return
}
