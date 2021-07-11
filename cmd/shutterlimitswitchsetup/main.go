package main

import (
	"fmt"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/balazsgrill/hass"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Main struct {
	app        fyne.App
	mainwindow fyne.Window
	client     mqtt.Client

	statusLabel   binding.String
	positionLabel binding.String
	coverSelect   *widget.Select
	data          map[string]*hass.Cover

	current *hass.Cover
}

func ConfigureClientOptions(p fyne.Preferences) *mqtt.ClientOptions {
	opts := mqtt.NewClientOptions().AddBroker(p.String("mqtt.host")).SetAutoReconnect(true)
	user := p.String("mqtt.user")
	pass := p.String("mqtt.pass")
	if user != "" {
		opts = opts.SetUsername(user)
	}
	if pass != "" {
		opts = opts.SetPassword(pass)
	}

	return opts
}

func (instance *Main) connect() {
	if instance.client == nil {
		opts := ConfigureClientOptions(instance.app.Preferences())
		opts.SetOnConnectHandler(func(c mqtt.Client) {
			instance.statusLabel.Set("Connected")
		})
		instance.client = mqtt.NewClient(opts)
	} else {
		// TODO reconfigure client?
	}
	for !instance.client.IsConnected() {
		token := instance.client.Connect()
		token.Wait()
		err := token.Error()
		if err != nil {
			dialog.ShowInformation("Error", err.Error(), instance.mainwindow)
		} else {
			errs := make(chan error)
			// TODO handle errors in log page instead
			hass.Discover(instance.client, instance, errs)
			go func() {
				for err = range errs {
					log.Println(err)
				}
			}()
		}
	}
}

func SettingsDialogContent(p fyne.Preferences) fyne.CanvasObject {
	host := widget.NewEntryWithData(binding.BindPreferenceString("mqtt.host", p))
	clientID := widget.NewEntryWithData(binding.BindPreferenceString("mqtt.clientid", p))
	user := widget.NewEntryWithData(binding.BindPreferenceString("mqtt.user", p))
	pass := widget.NewEntryWithData(binding.BindPreferenceString("mqtt.pass", p))
	return widget.NewForm(
		widget.NewFormItem("Broker", host),
		widget.NewFormItem("Client ID", clientID),
		widget.NewFormItem("Username", user),
		widget.NewFormItem("Password", pass),
	)
}

func (instance *Main) ConsumeCover(c *hass.Cover, nodeID string, objectID string) {
	_, exists := instance.data[c.Name]
	instance.data[c.Name] = c
	if !exists {
		instance.coverSelect.Options = append(instance.coverSelect.Options, c.Name)
	}
	instance.coverSelect.Refresh()
	log.Println(objectID)
}
func (instance *Main) ConsumeDInput(c *hass.DInput, nodeID string, objectID string) {}
func (instance *Main) ConsumeHVAC(c *hass.HVAC, nodeID string, objectID string)     {}
func (instance *Main) ConsumeSwitch(c *hass.Switch, nodeID string, objectID string) {}
func (instance *Main) ConsumeSensor(c *hass.Sensor, nodeID string, objectID string) {}
func (instance *Main) ConsumeLight(c *hass.Light, nodeID string, objectID string)   {}

func (instance *Main) selectChanged(key string) {
	if instance.current != nil {
		instance.client.Unsubscribe(instance.current.PositionTopic)
	}
	instance.current = instance.data[key]
	instance.client.Subscribe(instance.current.PositionTopic, 0, func(c mqtt.Client, m mqtt.Message) {
		instance.positionLabel.Set(string(m.Payload()) + " / " + fmt.Sprintf("%d", instance.current.PositionOpen))
	})
}

func (instance *Main) up() {
	if instance.current != nil {
		instance.client.Publish(instance.current.CommandTopic, 0, false, "10")
	}
}

func (instance *Main) stop() {
	if instance.current != nil {
		instance.client.Publish(instance.current.CommandTopic, 0, false, "0")
	}
}

func (instance *Main) down() {
	if instance.current != nil {
		instance.client.Publish(instance.current.CommandTopic, 0, false, "-10")
	}
}

func (instance *Main) send(data string) {
	if instance.current != nil {
		instance.client.Publish(instance.current.CommandTopic, 0, false, data)
	}
}

func main() {
	app := &Main{
		app:           app.NewWithID("WscgoClient"),
		statusLabel:   binding.NewString(),
		positionLabel: binding.NewString(),
		data:          make(map[string]*hass.Cover),
	}
	app.mainwindow = app.app.NewWindow("Shutter Limit Switch Setup")
	settingsdialog := SettingsDialogContent(app.app.Preferences())
	app.coverSelect = widget.NewSelect([]string{}, app.selectChanged)

	numCmdStr := binding.NewString()

	app.mainwindow.SetContent(container.NewVBox(
		widget.NewLabelWithData(app.statusLabel),
		widget.NewButton("Settings", func() {
			dialog.ShowCustom("Settings", "Apply", settingsdialog, app.mainwindow)
		}),
		widget.NewButton("Connect", func() {
			app.connect()
		}),
		app.coverSelect,
		widget.NewLabelWithData(app.positionLabel),
		widget.NewButton("Up", app.up),
		widget.NewButton("Stop", app.stop),
		widget.NewButton("Down", app.down),

		widget.NewEntryWithData(numCmdStr),
		widget.NewButton("Send", func() {
			v, err := numCmdStr.Get()
			if err == nil {
				app.send(v)
			}
		}),
	))

	app.mainwindow.Resize(fyne.NewSize(400, 700))
	app.mainwindow.ShowAndRun()
}
