package bolt_db

import (
	"bytes"
	"encoding/gob"
	"fmt"
)

func defaultEnc(s any) ([]byte, error) {
	if s == nil {
		return nil, fmt.Errorf("cannot encode nil value")
	}
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(s)
	if err != nil {
		return nil, fmt.Errorf("failed to encode struct: %v", err)
	}
	return buf.Bytes(), nil
}

func defaultDec(data []byte, s any) error {
	if len(data) == 0 {
		return fmt.Errorf("data is empty")
	}
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	err := dec.Decode(s)
	if err != nil {
		return fmt.Errorf("failed to decode bytes: %v", err)
	}
	return nil
}
