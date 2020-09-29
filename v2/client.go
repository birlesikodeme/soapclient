package v2

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"
)

// against "unused imports"
var _ time.Time
var _ xml.Name

type ResponseSOAPEnvelope struct {
	XMLName xml.Name `xml:"Envelope"`
	Ns1     string   `xml:"xmlns:ns1,attr,omitempty"`
	Ns2     string   `xml:"xmlns:ns2,attr,omitempty"`
	Ns3     string   `xml:"xmlns:ns3,attr,omitempty"`

	Header ResponseSOAPHeader
	Body   ResponseSOAPBody
}

type ResponseSOAPHeader struct {
	XMLName xml.Name `xml:"Header"`

	Header interface{}
}

type ResponseSOAPBody struct {
	XMLName xml.Name `xml:"Body"`

	Fault   *Fault      `xml:",omitempty"`
	Content interface{} `xml:",omitempty"`
}

func (b *ResponseSOAPBody) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	if b.Content == nil {
		return xml.UnmarshalError("Content must be a pointer to a struct")
	}

	var (
		token    xml.Token
		err      error
		consumed bool
	)

Loop:
	for {
		if token, err = d.Token(); err != nil {
			return err
		}

		if token == nil {
			break
		}

		switch se := token.(type) {
		case xml.StartElement:
			if consumed {
				return xml.UnmarshalError("Found multiple elements inside SOAP body; not wrapped-document/literal WS-I compliant")
			} else if se.Name.Space == "http://schemas.xmlsoap.org/soap/envelope/" && se.Name.Local == "Fault" {
				b.Fault = &Fault{}
				b.Content = nil

				err = d.DecodeElement(b.Fault, &se)
				if err != nil {
					return err
				}

				consumed = true
			} else {
				if err = d.DecodeElement(b.Content, &se); err != nil {
					return err
				}

				consumed = true
			}
		case xml.EndElement:
			break Loop
		}
	}

	return nil
}

func Parse(data []byte, v interface{}) error {
	env := new(ResponseSOAPEnvelope)
	env.Body = ResponseSOAPBody{Content: v}
	if err := xml.Unmarshal(data, env); err != nil {
		return err
	}
	fault := env.Body.Fault
	if fault != nil {
		return fault
	}
	return nil
}

func formatXML(data []byte) ([]byte, error) {
	b := &bytes.Buffer{}
	decoder := xml.NewDecoder(bytes.NewReader(data))
	encoder := xml.NewEncoder(b)
	encoder.Indent("", "  ")
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			encoder.Flush()
			return b.Bytes(), nil
		}
		if err != nil {
			return nil, err
		}
		err = encoder.EncodeToken(token)
		if err != nil {
			return nil, err
		}
	}
}

func debugPrintXml(info string, data []byte) {
	fmt.Println()
	fmt.Println("************************************************************************************")
	fmt.Println(info)
	b, _ := formatXML(data)
	fmt.Println(string(b))
	fmt.Println()
	fmt.Println("************************************************************************************")
	fmt.Println()
}

// Client for making soap calls 
type Client struct {
	// 
	httpClient HttpClient

	//
	base  string
	debug bool

	// Basic Authentication if needed
	username string
	password string

	// OAuth Authentication
	bearerToken string

	// Metadata
	userAgent   string
	contentType string
	action      string
}

// Creates a new soap client for a single base address with given option functions
func NewClient(httpClient HttpClient, baseAddress string, options ...func(*Client) error) (*Client, error) {

	client := &Client{
		httpClient: httpClient,
		base:       baseAddress,
	}

	for _, option := range options {
		err := option(client)
		if err != nil {
			return nil, err
		}
	}

	return client, nil
}

// INTERFACES
type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// OPTIONS

// Activates debug mode
func DebugMode() func(*Client) error {
	return func(client *Client) error {
		client.debug = true
		return nil
	}
}

// Use basic authentication with given username and password
func DefaultBasicAuth(username string, password string) func(*Client) error {
	return func(client *Client) error {
		client.username = username
		client.password = password
		return nil
	}
}

// Sets user agent
func DefaultUserAgent(userAgent string) func(*Client) error {
	return func(client *Client) error {
		client.userAgent = userAgent
		return nil
	}
}

// Sets default content type
func DefaultContentType(contentType string) func(*Client) error {
	return func(client *Client) error {
		client.contentType = contentType
		return nil
	}
}

// Sets default soap action
func DefaultAction(action string) func(*Client) error {
	return func(client *Client) error {
		client.action = action
		return nil
	}
}

// Sets http client to use given token
func DefaultBearerToken(token string) func(*Client) error {
	return func(client *Client) error {
		client.bearerToken = token
		return nil
	}
}

// METHODS

// Make soap call and parses response into the given struct
func (c *Client) CallWithContext(ctx context.Context, soapReq *Request, response interface{}) error {
	buffer, err := soapReq.Serialize()
	if err != nil {
		return err
	}

	if c.debug {
		debugPrintXml("Request:", []byte(buffer.String()))
	}

	url := fmt.Sprintf("%s%s", c.base, soapReq.path)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, buffer)
	if err != nil {
		return err
	}

	if soapReq.contentType != "" {
		httpReq.Header.Add("Content-Type", soapReq.contentType)
	} else if c.contentType != "" {
		httpReq.Header.Add("Content-Type", c.contentType)
	} else {
		httpReq.Header.Add("Content-Type", "text/xml; charset=\"utf-8\"")
	}

	if soapReq.action != "" {
		httpReq.Header.Add("SOAPAction", soapReq.action)
	} else if c.action != "" {
		httpReq.Header.Add("SOAPAction", c.action)
	}

	if soapReq.userAgent != "" {
		httpReq.Header.Set("User-Agent", soapReq.userAgent)
	} else if c.userAgent != "" {
		httpReq.Header.Add("User-Agent", c.userAgent)
	} else {
		httpReq.Header.Set("User-Agent", "Go")
	}

	httpReq.Close = true

	if soapReq.username != "" {
		httpReq.SetBasicAuth(soapReq.username, soapReq.password)
	} else if c.username != "" {
		httpReq.SetBasicAuth(c.username, c.password)
	}

	if soapReq.bearerToken != "" {
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", soapReq.bearerToken))
	} else if c.bearerToken != "" {
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.bearerToken))
	}

	res, err := c.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	rawbody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if len(rawbody) == 0 {
		// log.Println("empty response")
		return nil
	}

	if c.debug {
		debugPrintXml("Response:", rawbody)
	}

	if err := Parse(rawbody, response); err != nil {
		return err
	}
	return nil
}

func (c *Client) Call(request *Request, response interface{}) error {
	return c.CallWithContext(context.Background(), request, response)
}
