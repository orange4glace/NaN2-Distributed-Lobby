package netserver

import (
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestConnection(t *testing.T) {
	cnt := 0
	sc := 0
	tl := 0
	uk := 0
	ua := 0
	for i := 0; i < 10000; i++ {

		go func() {
			res, err := request("testuser", "testpassword")
			if err != nil {
				fmt.Println("### Error", err.Error())
				return
			}
			splits := strings.SplitN(res, " ", 3)
			userid := splits[0]
			token := splits[1]

			tcpAddr, err := net.ResolveTCPAddr("tcp", "localhost:6666")
			if err != nil {
				fmt.Println("### Error", err.Error())
				return
			}

			conn, err := net.DialTCP("tcp", nil, tcpAddr)
			if err != nil {
				fmt.Println("### Error", err.Error())
				return
			}
			cnt++
			go func() {
				_, err = conn.Write([]byte(userid + " " + token))
				if err != nil {
					fmt.Println("### Error", err.Error())
					return
				}
				time.Sleep(1 * time.Second)
			}()

			reply := make([]byte, 1024)

			issc := false
			isua := false
			for {
				len, err := conn.Read(reply)
				if err != nil {
					if err == io.EOF && !issc && !isua {
						tl++
						return
					}
					if !issc && !isua {
						println(err.Error())
						uk++
					}
					return
				}
				body := string(reply[:len])
				if strings.Compare(body, "session changed") == 0 {
					sc++
					issc = true
				}
				if strings.Compare(body, "401 unauthorized") == 0 {
					ua++
					isua = true
				}
			}
		}()
		time.Sleep(1 * time.Millisecond)
	}
	time.Sleep(3 * time.Second)
	println("cnt:", cnt)
	println("session changed:", sc)
	println("time out:", tl)
	println("unknown error:", uk)
	println("unauthorized:", ua)
	time.Sleep(100000 * time.Second)
}

func request(userid, password string) (string, error) {
	resp, err := http.PostForm("http://localhost:5555/authenticate", url.Values{"userid": {userid}, "password": {password}})
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	// Response 체크.
	respBody, err := ioutil.ReadAll(resp.Body)
	if err == nil {
		str := string(respBody)
		return str, nil
	}
	return "", nil
}
