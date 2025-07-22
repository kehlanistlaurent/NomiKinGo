// Nomi.go (Patched for RAW JSON Logging)
package NomiKin

import (
    "bytes"
    "encoding/json"
    "fmt"
    "log"
    "io"
    "net/http"
    "strings"
)

type Nomi struct {
    Uuid string
    Name string
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
    Text string
}

type NomiSentMessageContainer struct {
    SentMessage NomiMessage
}

type NomiReplyMessage struct {
    Text     string `json:"text"`
    ImageUrl string `json:"imageUrl"`
}

type NomiReplyMessageContainer struct {
    ReplyMessage NomiReplyMessage `json:"replyMessage"`
}

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
        var errorResult map[string]interface{}
        if err := json.Unmarshal(responseBody, &errorResult); err != nil {
            return nil, fmt.Errorf("Error unmarshalling API error response: %v\n%v", err, string(responseBody))
        }
        return nil, fmt.Errorf("Error response from Nomi API\n Error Code: %v\n Response Body: %v\n", resp.StatusCode, string(responseBody))
    }

    return responseBody, nil
}

func (nomi *NomiKin) RequestNomiRoomReply(roomId *string, nomiId *string) (string, error) {
    bodyMap := map[string]string{
        "nomiUuid": *nomiId,
    }

    messageSendUrl := NomiUrlComponents["RoomReply"][0] + "/" + *roomId + "/" + NomiUrlComponents["RoomReply"][1]
    response, err := nomi.ApiCall(messageSendUrl, "Post", bodyMap)
    if err != nil {
        log.Printf("Error from API call: %v", err.Error())
        return "", err
    }

    // 👇 LOG RAW API RESPONSE
    log.Printf("🔎 RAW Nomi API response: %s", string(response))

    var result NomiReplyMessageContainer
    if err := json.Unmarshal(response, &result); err != nil {
        log.Printf("Error parsing Nomi response: %v", err)
        return "", err
    }

    log.Printf("Received Message from Nomi %v to room %s: %v", nomi.CompanionId, *roomId, result.ReplyMessage.Text)

    if result.ReplyMessage.ImageUrl != "" {
        log.Printf("Received Image from Nomi %v: %v", nomi.CompanionId, result.ReplyMessage.ImageUrl)
        return fmt.Sprintf("%s||%s", result.ReplyMessage.Text, result.ReplyMessage.ImageUrl), nil
    }

    return result.ReplyMessage.Text, nil
}
