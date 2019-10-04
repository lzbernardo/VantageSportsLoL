package cli

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/howeyc/gopass"
)

// ReadPipedInput returns the slice of bytes piped into this command, or else
// an empty byte slice (with nil error).
func ReadPipedInput() ([]byte, error) {
	info, err := os.Stdin.Stat()
	if err != nil {
		return nil, err
	}
	pipedInput := info.Mode()&os.ModeNamedPipe != 0
	if info.Size() == 0 && !pipedInput {
		return []byte{}, nil
	}
	return ioutil.ReadAll(os.Stdin)
}

// ReadString prompts the user for a string.
func ReadString(prompt string) string {
	fmt.Printf("%s: ", prompt)
	r := bufio.NewReader(os.Stdin)
	str, _ := r.ReadString('\n')
	return strings.TrimSpace(str)
}

// ReadPassword allows the user to input a string without echoing into the
// shell.
func ReadPassword(prompt string) (string, error) {
	fmt.Printf("%s: ", prompt)
	pwd, err := gopass.GetPasswd()
	return string(pwd), err
}

// ReadStrings prompts the user for a list of strings.
func ReadStrings(prompt string) []string {
	str := ReadString(prompt)
	if str == "" {
		return []string{}
	}
	res := strings.Split(str, ",")
	for i := range res {
		res[i] = strings.TrimSpace(res[i])
	}
	return res
}

// ReadBool prompts the user for a boolean value.
func ReadBool(prompt string) bool {
	str := strings.ToLower(ReadString(prompt))
	return str == "t" || str == "true"
}
