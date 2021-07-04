module github.com/balazsgrill/wscgoclient

go 1.16

replace github.com/balazsgrill/hass => ../hass

require (
	fyne.io/fyne/v2 v2.0.3
	github.com/balazsgrill/hass v0.0.1
	github.com/eclipse/paho.mqtt.golang v1.3.5
)
