package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type ListSegmentPortsRequest struct {
	SegmentId string `json:"segment_id"`
}

type ListSegmentPortsResponse struct {
	ResultCount int           `json:"result_count"`
	Results     []SegmentPort `json:"results"`
}

type PatchSegmentPortRequest struct {
	SegmentId   string      `json:"segment_id"`
	PortId      string      `json:"port_id"`
	SegmentPort SegmentPort `json:"segment_port"`
}

type PortAddressBindingEntry struct {
	IpAddress  string `json:"ip_address"`
	MacAddress string `json:"mac_address"`
	VlanId     string `json:"vlan_id"`
}

type PortAttachment struct {
	AllocateAddresses string `json:"allocate_addresses"`
	AppId             string `json:"app_id"`
	ContextId         string `json:"context_id"`
	Id                string `json:"id"`
	TrafficTag        string `json:"traffic_tag"`
	Type              string `json:"type"`
}

type SegmentPort struct {
	AddressBindings PortAddressBindingEntry `json:"address_bindings"`
	AdminState      string                  `json:"admin_state"`
	Attachment      PortAttachment          `json:"attachment"`
	Description     string                  `json:"description"`
	DisplayName     string                  `json:"display_name"`
	Id              string                  `json:"id"`
	ResourceType    string                  `json:"resource_type"`
}

// RequestEditorFn  is the function signature for the RequestEditor callback function
type RequestEditorFn func(ctx context.Context, req *http.Request) error

// Doer performs HTTP requests.
//
// The standard http.Client implements this interface.
type HttpRequestDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type Client struct {
	Server string

	Username string

	Password string

	Client HttpRequestDoer

	RequestEditors []RequestEditorFn
}

type ClientOption func(*Client) error

func NewClient(server string, username string, password string, opts ...ClientOption) (*Client, error) {
	// create a client with sane default values
	client := Client{
		Server:   server,
		Username: username,
		Password: password,
	}
	// mutate client and add all optional params
	for _, o := range opts {
		if err := o(&client); err != nil {
			return nil, err
		}
	}
	// create httpClient, if not already present
	if client.Client == nil {
		client.Client = &http.Client{}
	}
	return &client, nil
}

// WithHTTPClient allows overriding the default Doer, which is
// automatically created using http.Client. This is useful for tests.
func WithHTTPClient(doer HttpRequestDoer) ClientOption {
	return func(c *Client) error {
		c.Client = doer
		return nil
	}
}

// WithRequestEditorFn allows setting up a callback function, which will be
// called right before sending the request. This can be used to mutate the request.
func WithRequestEditorFn(fn RequestEditorFn) ClientOption {
	return func(c *Client) error {
		c.RequestEditors = append(c.RequestEditors, fn)
		return nil
	}
}

func (c *Client) applyEditors(ctx context.Context, req *http.Request, additionalEditors []RequestEditorFn) error {
	for _, r := range c.RequestEditors {
		if err := r(ctx, req); err != nil {
			return err
		}
	}
	for _, r := range additionalEditors {
		if err := r(ctx, req); err != nil {
			return err
		}
	}
	return nil
}

type ClientInterface interface {
	DeleteSegmentPort(string) (*http.Response, error)
	ListSegmentPorts(string) (*ListSegmentPortsResponse, error)
	GetSegmentPort(string, string) (*SegmentPort, error)
	PatchSegmentPort(string, string) (*bool, error)
}

func (c *Client) DeleteSegmentPort(ctx context.Context, segment_id string, port_id string, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewDeleteSegmentPortRequest(c.Server, segment_id, port_id)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func NewDeleteSegmentPortRequest(server string, segment_id string, port_id string) (*http.Request, error) {
	var err error

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/policy/api/v1/infra/segments/%s/ports/%s", segment_id, port_id)
	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("DELETE", queryURL.String(), nil)
	if err != nil {
		return nil, err
	}

	return req, nil
}

func (c *Client) ListSegmentPorts(ctx context.Context, segment_id string, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewListSegmentPortsRequest(c.Server, c.Username, c.Password, segment_id)
	if err != nil {
		return nil, err
	}

	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func NewListSegmentPortsRequest(server string, user string, pass string, segment_id string) (*http.Request, error) {
	var err error

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := "/policy/api/v1/infra/segments/" + segment_id + "/ports"
	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodGet, queryURL.String(), nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(user, pass)

	return req, nil
}

func (c *Client) GetSegmentPort(ctx context.Context, segment_id string, port_id string, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewGetSegmentPortRequest(c.Server, c.Username, c.Password, segment_id, port_id)

	if err != nil {
		return nil, err
	}

	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func NewGetSegmentPortRequest(server string, user string, pass string, segment_id string, port_id string) (*http.Request, error) {
	var err error

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := "/policy/api/v1/infra/segments/" + segment_id + "/ports/" + port_id
	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodGet, queryURL.String(), nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(user, pass)

	return req, nil
}

func (c *Client) PatchSegmentPort(ctx context.Context, body PatchSegmentPortRequest, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewPatchSegmentPortRequest(c.Server, c.Username, c.Password, body)
	if err != nil {
		return nil, err
	}

	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func NewPatchSegmentPortRequest(server string, user string, pass string, body PatchSegmentPortRequest) (*http.Request, error) {
	var err error

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := "/policy/api/v1/infra/segments/" + body.SegmentId + "/ports/" + body.PortId
	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	var bodyReader io.Reader
	buf, err := json.Marshal(body.SegmentPort)
	if err != nil {
		return nil, err
	}
	bodyReader = bytes.NewReader(buf)

	req, err := http.NewRequest(http.MethodPatch, queryURL.String(), bodyReader)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(user, pass)
	req.Header.Add("Content-Type", "application/json")

	return req, nil
}
