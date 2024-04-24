package netfunnel

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	ApiEndpoint   string        // the netfunnel server endpoint. you should know that the netfunnel endpoint is different by the api server.
	RetryInterval time.Duration // if waitlist is full, delay time to check again. (recommend: 1s)
}

// Check waitlist status from the server with the key.
//
// if success, return 200.
func (t *Client) CheckWaitlist(key string) (int, error) {
	// TODO: chkEnter에 대한 정확한 params 값 확인 필요
	params := url.Values{}
	params.Add("opcode", "5002")
	params.Add("key", key)
	params.Add("nfid", "0")
	params.Add("prefix", "NetFunnel.gRtype=5002;")
	params.Add("ttl", "0")
	params.Add("js", "yes")

	apiurl := appendStr(t.ApiEndpoint, "?", params.Encode(), "&", fmt.Sprint(time.Now().UnixMilli()))
	resp, err := http.Get(apiurl)
	if err != nil {
		return -1, errors.New("failed to connect to the netfunnel server. (in CheckWaitlist)")
	}

	return resp.StatusCode, nil
}

// The netfunnel ticket struct.
//
// if you need, you can to get ticket from the server by calling GetTicket() method.
// and if you can requesting another api server (using netfunnel), maybe you need this ID of the ticket.
type Ticket struct {
	Id    string
	Ip    string
	Key   string
	Nnext int
	Nwait int
	Port  int
	Tps   int
	Ttl   int
}

// Getting ticket from the server
func (t *Client) GetTicket() (Ticket, error) {
	params := url.Values{}
	params.Add("opcode", "5101")
	params.Add("nfid", "0")
	params.Add("prefix", "NetFunnel.gRtype=5101;")
	params.Add("sid", "service_1")
	params.Add("aid", "act_1")
	params.Add("js", "yes")

	// Requesting to the server
	apiurl := appendStr(t.ApiEndpoint, "?", params.Encode(), "&", fmt.Sprint(time.Now().UnixMilli()))
	resp, err := http.Get(apiurl)
	if err != nil {
		return Ticket{}, errors.New("failed to connect to the netfunnel server")
	}

	// Reading response body
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return Ticket{}, errors.New("failed to read response body")
	}

	// parsing ticket from response body
	ticket, err := parseTicketStr(string(respBytes))
	if err != nil {
		return Ticket{}, err
	}

	// Check HTTP Status Code
	switch resp.StatusCode {
	case 200:
		// Success (waitlist is empty)
		break
	case 201:
		for {
			resHttpCode, err := t.CheckWaitlist(ticket.Key)
			if err != nil {
				return Ticket{}, err
			}
			if resHttpCode == 200 {
				// Success (waited, but now it's your turn)
				break
			}
			time.Sleep(t.RetryInterval)
		}
	default:
		return Ticket{}, fmt.Errorf("unexpected HTTP Status Code. (HTTP %d)", resp.StatusCode)
	}

	return ticket, nil
}

// Dispatching ticket to the server
func (t *Client) DispatchTicket(ticket *Ticket) error {
	params := url.Values{}
	params.Add("opcode", "5004")
	params.Add("key", ticket.Key)
	params.Add("nfid", "0")
	params.Add("prefix", "NetFunnel.gRtype=5004;")
	params.Add("sid", "service_1")
	params.Add("aid", "act_1")
	params.Add("js", "yes")

	// Requesting to the server
	apiurl := appendStr(t.ApiEndpoint, "?", params.Encode(), "&", fmt.Sprint(time.Now().UnixMilli()))
	resp, err := http.Get(apiurl)
	if err != nil {
		return errors.New("failed to connect to the netfunnel server")
	}

	switch resp.StatusCode {
	case 200:
		return nil
	default:
		return fmt.Errorf("unexpected HTTP Status Code. (HTTP %d)", resp.StatusCode)
	}
}

// Parsing netfunnel ticket response body. it return Ticket.
func parseTicketStr(body string) (Ticket, error) {
	scriptVars := make(map[string]string)

	// parsing response body. it expects to be a javascript variable declaration.
	for _, v := range strings.Split(body, ";") {
		kvArray := strings.SplitN(v, "=", 2)
		scriptVars[strings.Trim(kvArray[0], " ")] = strings.Trim(kvArray[1], " ")
	}

	// check NetFunnel.gControl.result exists
	if scriptVars["NetFunnel.gControl.result"] == "" {
		return Ticket{}, errors.New("failed to parse response body: Cannot find 'NetFunnel.gControl.result' key")
	}

	id := strings.Trim(scriptVars["NetFunnel.gControl.result"], "'") // remove single quotes

	// parsing NetFunnel.gControl.result
	// example: <int>:<int>:ip=nf.example.com&key=<string>&nnext=<int>&nwait=<int>&port=<int>&tps=<int>&ttl=<int>
	val, err := url.ParseQuery(strings.SplitN(id, ":", 2)[2]) // we need only "ip=nf.example.com&key=<string>&nnext=<int>&nwait=<int>&port=<int>&tps=<int>&ttl=<int>"
	if err != nil {
		return Ticket{}, fmt.Errorf("failed to parse response body (url.ParseQuery failed): %s", err)
	}

	val_nnext, err := strconv.Atoi(val.Get("nnext"))
	if err != nil {
		return Ticket{}, fmt.Errorf("failed to parse response body (Atoi Failed, in nnext): %s", err)
	}

	val_nwait, err := strconv.Atoi(val.Get("nwait"))
	if err != nil {
		return Ticket{}, fmt.Errorf("failed to parse response body (Atoi Failed, in nwait): %s", err)
	}

	val_port, err := strconv.Atoi(val.Get("port"))
	if err != nil {
		return Ticket{}, fmt.Errorf("failed to parse response body (Atoi Failed, in port): %s", err)
	}

	val_tps, err := strconv.Atoi(val.Get("tps"))
	if err != nil {
		return Ticket{}, fmt.Errorf("failed to parse response body (Atoi Failed, in tps): %s", err)
	}

	val_ttl, err := strconv.Atoi(val.Get("ttl"))
	if err != nil {
		return Ticket{}, fmt.Errorf("failed to parse response body (Atoi Failed, in ttl): %s", err)
	}

	return Ticket{
		Id:    id,
		Ip:    val.Get("ip"),
		Key:   val.Get("key"),
		Nnext: val_nnext,
		Nwait: val_nwait,
		Port:  val_port,
		Tps:   val_tps,
		Ttl:   val_ttl,
	}, nil
}
