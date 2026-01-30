package api

import (
	"errors"
	"net/http"
)

var (
	UnauthorizedError = errors.New("unauthorized")
)

func MapResponseToError(response *http.Response, err error) error {
	if response == nil {
		return errors.Join(errors.New("response is nil"), err)
	}

	var mappedErr error
	switch response.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		mappedErr = UnauthorizedError
	}

	return errors.Join(mappedErr, err)
}
