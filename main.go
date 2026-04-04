package main 
    
import "access-control-status-bridge/messages"
import "access-control-status-bridge/export"
import "context"
import "fmt"
import mqtt "github.com/eclipse/paho.mqtt.golang"
import "io/ioutil"
import "crypto/x509"
import "crypto/tls"
import "log"
import "time"
import "strings"
import "os"
import "os/signal"
import "syscall"
import "encoding/json"

const (
    broker   = "ssl://a1j7mrp8z8zjsh-ats.iot.eu-west-2.amazonaws.com:8883"
    clientID = "status-application-test"
)

func main() {
    fmt.Println("Access Control Status Bridge")

    cert, err := tls.LoadX509KeyPair(
        "2e4b0db865633c4dac45e7d2b573519d3bb8122fa70b102204b79132bb59904c-certificate.pem.crt",
        "2e4b0db865633c4dac45e7d2b573519d3bb8122fa70b102204b79132bb59904c-private.pem.key",
    )

    if err != nil {
        log.Fatal(err)
    }

    caCertPool := x509.NewCertPool()
    pemData, err := ioutil.ReadFile("AmazonRootCA1.pem")

    if err != nil {
        log.Fatal(err)
    }

    caCertPool.AppendCertsFromPEM(pemData)

    tlsConfig := &tls.Config{
        RootCAs: caCertPool,
        Certificates: []tls.Certificate{cert},
    }

    opts := mqtt.NewClientOptions()
    opts.AddBroker(broker)
    opts.SetClientID(clientID)
    opts.SetTLSConfig(tlsConfig)

    c := mqtt.NewClient(opts)

    if token := c.Connect(); token.Wait() && token.Error() != nil {
        panic(token.Error())
    }

    ctx, cancel := context.WithCancel(context.Background())

    modeHandler := messages.CreateModeHandler(ctx)

    subscribeToVersionMessage(c, "acs/message/mode/#", modeHandler)
    go modeHandler.Poll()
    go export.ModeToPrometheus(ctx, modeHandler.Export)

    go func () {
        for {
            select {
                case mode := <- modeHandler.Outbox:
                    buff, err := json.Marshal(mode)
                    if err == nil && false {
                        fmt.Println(string(buff))
                    }

                case <- ctx.Done():
                    return
            }
        }
    }()

    powerHandler := messages.CreatePowerHandler(ctx)
    subscribeToVersionMessage(c, "acs/message/power/#", powerHandler)
    go powerHandler.Poll()
    go export.PowerToPrometheus(ctx, powerHandler.Export)


    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
    <-sigChan

    cancel()
    c.Disconnect(250)
}

func subscribeToVersionMessage(client mqtt.Client, topic string, messageHandler messages.AccessMessageHandler) mqtt.Token {

    return client.Subscribe(topic, 0, func (client mqtt.Client, msg mqtt.Message) {
        topicParts := strings.Split(msg.Topic(), "/")
        //fmt.Println(msg.Topic())
        rawMessage := messages.RawMessage{
            Payload : msg.Payload(),
            Received : time.Now(),
            Version : topicParts[3],
        }

        messageHandler.In() <- rawMessage
    })
}
