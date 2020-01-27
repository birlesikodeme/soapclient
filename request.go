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

	path string `xml:"-"`
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
// If no attribute with "soapenv" is given, default value is used
func NewRequest(path string, body interface{}, header []interface{}, action string, attributes ...xml.Attr) *Request {

	// If no attribute with name soapenv, set default value
	found := false
	for _, attribute := range attributes {
		if attribute.Name.Local == "soapenv" {
			found = true
		}
	}

	if !found {
		attributes = append(attributes, xml.Attr{
			Name:  xml.Name{Local: "soapenv",},
			Value: "http://schemas.xmlsoap.org/soap/envelope/",
		})
	}

	// Fill soap envelope

	envelope := Request{
		path:       path,
		Attributes: attributes,
	}

	// Only add header if it exists
	if header != nil {
		envelope.Header = RequestSOAPHeader{Header: header}
	}

	envelope.Body.Content = body

	return &envelope
}

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
