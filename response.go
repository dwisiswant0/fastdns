package fastdns

import (
	"time"

	"github.com/go-json-experiment/json"
	"github.com/miekg/dns"
)

// Response represents the result of a DNS query.
type Response struct {
	Error error `json:"error"`

	// Message is the DNS message received from the server.
	Message *dns.Msg `json:"message"`

	// RTT is the round-trip time for the DNS query.
	// It represents the duration from sending the query to receiving the
	// response.
	//
	// RTT is set to -1 when there is no response or the query fails.
	// RTT is set to 0 when the response is served from cache.
	// Otherwise, RTT is set to the actual time taken for the query.
	RTT time.Duration `json:"rtt"`

	// query is the original DNS query sent to the server.
	query *dns.Msg `json:"-"`
}

// String returns a string representation of the [Result].
// If there is an error, it returns the error message.
// If there is no error and the Message is nil, it returns an empty string.
// If there is no error and the Message is not nil, it returns the string
// representation of the Message.
func (r *Response) String() string {
	if r.Error != nil {
		return r.Error.Error()
	}

	if r.Message == nil {
		return ""
	}

	return r.Message.String()
}

// Err returns the error associated with the [Result].
// If there is no error, it returns nil.
// If there is an error, it returns the error.
func (r *Response) Err() error {
	return r.Error
}

// IsError checks if the [Result] contains an error.
// It returns true if there is an error, and false otherwise.
// This is useful for checking if the DNS query was successful or not.
// If there is no error, it returns false.
// If there is an error, it returns true.
func (r *Response) IsError() bool {
	return r.Error != nil
}

// IsSuccess checks if the [Result] is successful.
// It returns true if there is no error and the Message is not nil.
// This is useful for checking if the DNS query was successful.
// If there is an error, it returns false.
// If the Message is nil, it returns false.
// If the Message is not nil and there is no error, it returns true.
func (r *Response) IsSuccess() bool {
	return r.Error == nil && r.Message != nil
}

// MarshalJSON marshals the [Result] into JSON format.
func (r *Response) MarshalJSON() ([]byte, error) {
	return json.Marshal(r)
}
