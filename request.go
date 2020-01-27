package soap

import (
	"bytes"
	"encoding/xml"
)

type Request struct {
	XMLName    xml.Name   `xml:"soapenv:Envelope"`
	Attributes []xml.Attr `xml:",attr,omitempty"`

	Header RequestSOAPHeader
	Body   RequestSOAPBody

	// Path of the soap action after the base address
	path string `xml:"-"`

	// Basic Authentication if needed
	username string `xml:"-"`
	password string `xml:"-"`

	// OAuth Authentication
	bearerToken string `xml:"-"`

	// Metadata
	userAgent   string `xml:"-"`
	contentType string `xml:"-"`
	action      string `xml:"-"`
}

type RequestSOAPEnvelope struct {
	XMLName xml.Name `xml:"soapenv:Envelope"`
	SoapENV string   `xml:"xmlns:soapenv,attr,omitempty"`
	Ns1     string   `xml:"xmlns:ns1,attr,omitempty"`
	Ns2     string   `xml:"xmlns:ns2,attr,omitempty"`
	Ns3     string   `xml:"xmlns:ns3,attr,omitempty"`

	Header RequestSOAPHeader
	Body   RequestSOAPBody
}

type RequestSOAPHeader struct {
	XMLName xml.Name `xml:"soapenv:Header"`

	Header []interface{}
}

type RequestSOAPBody struct {
	XMLName xml.Name `xml:"soapenv:Body"`

	Fault   *Fault      `xml:",omitempty"`
	Content interface{} `xml:",omitempty"`
}

// Creates a new soap request with given attributes
func NewRequest(path string, body interface{}, header []interface{}, options ...func(*Request) error) *Request {

	// Fill soap envelope
	envelope := Request{
		path: path,
		Attributes: []xml.Attr{
			// Set default attribute for soapenv
			{
				Name:  xml.Name{Local: "soapenv",},
				Value: "http://schemas.xmlsoap.org/soap/envelope/",
			},
		},
	}

	// Only add header if it exists
	if header != nil {
		envelope.Header = RequestSOAPHeader{Header: header}
	}

	envelope.Body.Content = body

	return &envelope
}

// OPTIONS

// Use basic authentication with given username and password
func BasicAuth(username string, password string) func(*Request) error {
	return func(request *Request) error {
		request.username = username
		request.password = password
		return nil
	}
}

// Sets user agent
func UserAgent(userAgent string) func(*Request) error {
	return func(request *Request) error {
		request.userAgent = userAgent
		return nil
	}
}

// Sets default content type
func ContentType(contentType string) func(*Request) error {
	return func(request *Request) error {
		request.contentType = contentType
		return nil
	}
}

// Sets default soap action
func Action(action string) func(*Request) error {
	return func(request *Request) error {
		request.action = action
		return nil
	}
}

// Sets http client to use given token
func BearerToken(token string) func(*Request) error {
	return func(request *Request) error {
		request.bearerToken = token
		return nil
	}
}

// Add new attributes to use with request. Default: [soapenv:"http://schemas.xmlsoap.org/soap/envelope/"]
func AddAttributes(attributes ...xml.Attr) func(*Request) error {

	// If no attribute with name soapenv, set default value
	found := false
	for _, attribute := range attributes {
		if attribute.Name.Local == "soapenv" {
			found = true
		}
	}

	return func(request *Request) error {

		requestAttributes := request.Attributes
		// Remove existing soapenv value
		if found {
			filtered := requestAttributes[:0]
			for _, attribute := range requestAttributes {
				if attribute.Name.Local != "soapenv" {
					filtered = append(filtered, attribute)
				}
			}
			requestAttributes = filtered
		}

		// Combine exiting attributes with new
		request.Attributes = append(requestAttributes, attributes...)
		return nil
	}
}

// METHODS
func (r Request) Serialize() (*bytes.Buffer, error) {
	buff := new(bytes.Buffer)

	encoder := xml.NewEncoder(buff)

	if err := encoder.Encode(r); err != nil {
		return nil, err
	}

	if err := encoder.Flush(); err != nil {
		return nil, err
	}

	return buff, nil
}
