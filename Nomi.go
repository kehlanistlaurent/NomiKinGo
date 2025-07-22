// Nomi.go (Surgical Patch: FetchRecentMessages + Polling)
package NomiKin

import (
    "bytes"
    "encoding/json"
    "fmt"
    "log"
    "io"
    "net/http"
    "strings"
    "time"
)

type Nomi struct {
    ApiKey      string
    CompanionId string
    LastMessageID string // track last message ID
}

type Room struct {
    Name string
    Uuid string
    Nomis []Nomi
}

type RoomContainer struct {
    Rooms []Room
}

type NomiMessage struct {
    Text     string `json:"text"`
    ImageUrl string `json:"imageUrl"`
    Id       string `json:"id"`
    Timestamp string `json:"timestamp"`
}

type RecentMessagesResponse struct {
    Messages []NomiMessage `json:"messages"`
}

// Existing ApiCall unchanged
func (nomi *NomiKin) ApiCall(endpoint string, method string, body interface{}) ([]byte, error) {
    method = strings.ToUpper(method)
    headers := map[string]string{
        "Authorization": nomi.ApiKey,
        "Content-Type": "application/json",
    }

    var jsonBody []byte
    var bodyReader io.Reader
    var err error

    if body != nil {
        jsonBody, err = json.Marshal(body)
        if err != nil {
            return nil, fmt.Errorf("Error constructing body: %v", err)
        }
        bodyReader = bytes.NewBuffer(jsonBody)
    }

    req, err := http.NewRequest(method, endpoint, bodyReader)
    if err != nil {
        return nil, fmt.Errorf("Error reading HTTP request: %v", err)
    }

    req.Header.Set("Authorization", headers["Authorization"])
    req.Header.Set("Content-Type", headers["Content-Type"])

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("Error making HTTP request: %v", err)
    }
    defer resp.Body.Close()

    responseBody, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("Error reading HTTP response: %v", err)
    }

    if resp.StatusCode < 200 || resp.StatusCode > 299 {
        return nil, fmt.Errorf("Error response from Nomi API\n Error Code: %v\n Response Body: %v\n", resp.StatusCode, string(responseBody))
    }

    return responseBody, nil
}

// 🔥 Fetch recent messages (probe API)
func (nomi *NomiKin) FetchRecentMessages(roomId string) ([]NomiMessage, error) {
    url := fmt.Sprintf("https://api.nomi.ai/v1/rooms/%s/messages", roomId)
    response, err := nomi.ApiCall(url, "GET", nil)
    if err != nil {
        log.Printf("Error fetching recent messages: %v", err)
        return nil, err
    }

    log.Printf("🔎 RAW FetchRecentMessages response: %s", string(response))

    var messagesResp RecentMessagesResponse
    if err := json.Unmarshal(response, &messagesResp); err != nil {
        log.Printf("Error parsing FetchRecentMessages response: %v", err)
        return nil, err
    }
    return messagesResp.Messages, nil
}

// 🔥 Start polling loop for new messages
func (nomi *NomiKin) StartPollingForNewMessages(roomId string, discordChannelID string, discordSession *discordgo.Session) {
    go func() {
        for {
            messages, err := nomi.FetchRecentMessages(roomId)
            if err == nil && len(messages) > 0 {
                latest := messages[len(messages)-1]
                if latest.Id != nomi.LastMessageID {
                    nomi.LastMessageID = latest.Id
                    log.Printf("📸 New message detected: %v", latest.Text)
                    
                    discordSession.ChannelMessageSend(discordChannelID, latest.Text)
                    if latest.ImageUrl != "" {
                        SendImageToDiscord(discordSession, discordChannelID, latest.ImageUrl)
                    }
                }
            }
            time.Sleep(10 * time.Second)
        }
    }()
}
