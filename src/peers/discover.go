package discover

import (
	"fmt"
	"io"
	// "net/http"
	"net/url"
)

func main() {
	baseUrl := "https://api.github.com/users/octocat"

	u, err := url.Parse(baseUrl)
	if err != nil {
		panic(err)
	}

	q := u.Query()
	q.Add("info_hash")
	q.Add("peer_id")
	q.Add("port")
	q.Add("uploaded")
	q.Add("downloaded")
	q.Add("left")
	q.Add("compact")

	body, err := io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}

	fmt.Println("Status code: ", res.StatusCode)
	fmt.Println("Body: ")
	fmt.Println(string(body))
}
