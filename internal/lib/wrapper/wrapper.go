package wrapper

import "fmt"

func Wrap(msg string, err error) error {
	if err == nil {
		return nil
	} else {
		return fmt.Errorf("%s: %w", msg, err)
	}
}
