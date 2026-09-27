package integration

import (
	"encoding/json"

	"github.com/pkg/errors"
)

func marshalJSON(v any, what string) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, errors.Wrap(err, "marshal "+what)
	}
	return b, nil
}
