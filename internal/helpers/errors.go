package helpers

import (
	"errors"
	"fmt"
)

var ErrSvcNotEnabled error = errors.New("service is not enabled")

func ErrListToErr(prefix string, errs []error) error {
	if len(errs) == 0 {
		return nil
	}

	var s string

	for _, e := range errs {
		if e != nil {
			if s == "" {
				s = fmt.Sprintf(": %s", e.Error())
			} else {
				s = fmt.Sprintf("%s, %s", s, e.Error())
			}
		}
	}

	return fmt.Errorf("%s%s", prefix, s)
}
