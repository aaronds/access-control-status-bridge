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

func main() {
    fmt.Println("Access Control Status Bridge")

    exportTlsConfig := getTlsContextFromEnv("EXPORT")
    subscribeTlsConfig := getTlsContextFromEnv("SUBSCRIBE")
    publishTlsConfig := getTlsContextFromEnv("PUBLISH")

    
    publishTopicMode := os.Getenv("BRIDGE_PUBLISH_TOPIC_MODE_PREFIX");

    if publishTopicMode == "" {
        publishTopicMode = "tool"
    }

    publishTopicEnvPm := os.Getenv("BRIDGE_PUBLISH_TOPIC_MODE_PREFIX");

    if publishTopicEnvPm == "" {
        publishTopicEnvPm = "env"
    }

    if exportTlsConfig != nil {
        prometheus.TlsClient = &http.Client{
            Transport : &http.Transport{
                TLSClientConfig : exportTlsConfig,
            },
        }
    }

    ctx, cancel := context.WithCancel(context.Background())

    publishOpts := mqtt.NewClientOptions()
    setMqttOpts(publishOpts, publishTlsConfig, "PUBLISH")

    var publishClient mqtt.Client = nil;
    var publishConnectToken mqtt.Token = nil;

    if len(publishOpts.Servers) > 0 {
        publishOpts.SetCleanSession(true)
        publishOpts.SetConnectionLostHandler(func (c mqtt.Client, err error) {
            log.Println(err)
        })
        publishClient = mqtt.NewClient(publishOpts)
        publishConnectToken = publishClient.Connect()
    }

    modeHandler := messages.CreateModeHandler(ctx)
    go modeHandler.Poll()
    go export.ModeToPrometheus(ctx, modeHandler.Export)
    if publishClient != nil {
        go modeHandler.Push(publishClient, publishTopicMode)
    }

    powerHandler := messages.CreatePowerHandler(ctx)
    go powerHandler.Poll()
    go export.PowerToPrometheus(ctx, powerHandler.Export)

    envPmHandler := messages.CreateEnvPmHandler(ctx)
    go envPmHandler.Poll()
    go export.EnvPmToPrometheus(ctx, envPmHandler.Export)

    if publishClient != nil {
        go envPmHandler.Push(publishClient, publishTopicEnvPm)
    }

    acsErrorHandler := messages.CreateAcsErrorHandler(ctx)
    go acsErrorHandler.Poll()
    go export.AcsErrorToPrometheus(ctx, acsErrorHandler.Export)

    subscribeOpts := mqtt.NewClientOptions()
    setMqttOpts(subscribeOpts, subscribeTlsConfig, "SUBSCRIBE")

    subscribeOpts.SetOnConnectHandler(func (c mqtt.Client) {
        subscribeToVersionMessage(c, "acs/message/error/#", acsErrorHandler);
        subscribeToVersionMessage(c, "env/message/pm/#", envPmHandler);
        subscribeToVersionMessage(c, "acs/message/power/#", powerHandler)
        subscribeToVersionMessage(c, "acs/message/mode/#", modeHandler)
    })

    subscribeOpts.SetConnectionLostHandler(func (c mqtt.Client, err error) {
        log.Println(err)
    })

    c := mqtt.NewClient(subscribeOpts)

    if token := c.Connect(); token.Wait() && token.Error() != nil {
        panic(token.Error())
    }

    if publishConnectToken != nil && publishConnectToken.Wait() && publishConnectToken.Error() != nil {
        log.Println(publishConnectToken.Error())
    }

    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
    <-sigChan

    cancel()
    c.Disconnect(250)

    if publishClient != nil && publishClient.IsConnected() {
        publishClient.Disconnect(250)
    }
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

func setMqttOpts(opts *mqtt.ClientOptions, tlsConfig *tls.Config, key string) *mqtt.ClientOptions {
    
    broker := os.Getenv("BRIDGE_" + key + "_BROKER")
    clientID := os.Getenv("BRIDGE_" + key + "_CLIENT_ID")
    username := os.Getenv("BRIDGE_" + key + "_USERNAME")
    password := os.Getenv("BRIDGE_" + key + "_PASSWORD")

    if clientID == "" {
        clientID = "access-control-status-bridge-" + key
    }

    opts.SetClientID(clientID)

    if broker != "" {
        opts.AddBroker(broker)
    }

    if username != "" {
        opts.SetUsername(username)
    }

    if password != "" {
        opts.SetPassword(password)
    }

    if tlsConfig != nil {
        opts.SetTLSConfig(tlsConfig)
    }

    return opts
}
