package main 
    
import "access-control-status-bridge/messages"
import "access-control-status-bridge/export"
import "access-control-status-bridge/prometheus"
import "context"
import "fmt"
import mqtt "github.com/eclipse/paho.mqtt.golang"
import "io/ioutil"
import "crypto/x509"
import "crypto/tls"
import "net/http"
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

    exportTlsConfig := getTlsContextFromEnv("EXPORT")
    subscribeTlsConfig := getTlsContextFromEnv("SUBSCRIBE")
    //publishTlsConfig := getTlsContextFromEnv("PUBLISH")

    subscribeBroker := os.Getenv("BRIDGE_SUBSCRIBE_BROKER")
    subscribeClientID := os.Getenv("BRIDGE_SUBSCRIBE_CLIENT_ID")

    if subscribeClientID == "" {
        subscribeClientID = "access-control-status-bridge-subscribe"
    }


    if exportTlsConfig != nil {
        prometheus.TlsClient = &http.Client{
            Transport : &http.Transport{
                TLSClientConfig : exportTlsConfig,
            },
        }
    }

    opts := mqtt.NewClientOptions()
    opts.AddBroker(subscribeBroker)
    opts.SetClientID(subscribeClientID)

    if subscribeTlsConfig != nil {
        opts.SetTLSConfig(subscribeTlsConfig)
    }

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

    envPmHandler := messages.CreateEnvPmHandler(ctx)
    subscribeToVersionMessage(c, "env/message/pm/#", envPmHandler);
    go envPmHandler.Poll()
    go export.EnvPmToPrometheus(ctx, envPmHandler.Export)

    acsErrorHandler := messages.CreateAcsErrorHandler(ctx)
    subscribeToVersionMessage(c, "acs/message/error/#", acsErrorHandler);
    go acsErrorHandler.Poll()
    go export.AcsErrorToPrometheus(ctx, acsErrorHandler.Export)

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

func getTlsContextFromEnv(key string) *tls.Config {

    var envTlsConfig *tls.Config = nil

    envTlsCert := os.Getenv("BRIDGE_" + key + "_TLS_CERT")
    envTlsKey := os.Getenv("BRIDGE_" + key + "_TLS_KEY")
    envTlsCA := os.Getenv("BRIDGE_" + key + "_TLS_CA")

    if envTlsCert != "" {

        cert, err := tls.LoadX509KeyPair(
            envTlsCert,
            envTlsKey,
        )

        if err != nil {
            log.Fatal(err)
        }

        caCertPool := x509.NewCertPool()
        pemData, err := ioutil.ReadFile(envTlsCA)

        if err != nil {
            log.Fatal(err)
        }

        caCertPool.AppendCertsFromPEM(pemData)

        envTlsConfig = &tls.Config{
            RootCAs: caCertPool,
            Certificates: []tls.Certificate{cert},
        }

        return envTlsConfig
    } else {
        return nil
    }
}
