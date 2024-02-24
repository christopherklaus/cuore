package integrations

import (
	"os"
)

func init() {
	os.Setenv("LOG_PATH", "messages.log")
}

// func TestAPIRequest(t *testing.T) {

// 	var session Sonos

// 	resp, err := session.SonosAPIRequest("https://api.ws.sonos.com/control/api/v1/households/Space/groups", "GET", nil)
// 	if err != nil {
// 		t.Fatalf("Failed to make request to Sonos API, %v", err)
// 	}

// 	fmt.Println(resp)

// 	// if !strings.Contains("", "") {
// 	// 	t.Fatalf("Expected empty string to contain empty string")
// 	// }
// }
