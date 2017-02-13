package backends

import (
	"github.com/flashmob/go-guerrilla/envelope"
)

type debuggerConfig struct {
	LogReceivedMails bool `json:"log_received_mails"`
}

func Debugger() Decorator {
	var config *debuggerConfig
	initFunc := Initialize(func(backendConfig BackendConfig) error {
		configType := baseConfig(&debuggerConfig{})
		bcfg, err := ab.extractConfig(backendConfig, configType)
		if err != nil {
			return err
		}
		config = bcfg.(*debuggerConfig)
		return nil
	})
	Service.AddInitializer(initFunc)
	return func(c Processor) Processor {
		return ProcessorFunc(func(e *envelope.Envelope) (BackendResult, error) {
			if config.LogReceivedMails {
				mainlog.Infof("Mail from: %s / to: %v", e.MailFrom.String(), e.RcptTo)
				mainlog.Info("So, Headers are:", e.Header)
			}
			// continue to the next Processor in the decorator chain
			return c.Process(e)
		})
	}
}
