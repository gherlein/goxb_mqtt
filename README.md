# goxb

This project uses libusb and implements a native XBox360(tm) controller reader in go.  Events 
are read from the controller and written to an MQTT message broker.

This tool requires that libusb is installed.

## Usage

### Linux

```
sudo ./goxb_mqtt --deadzone=512 --broker="tcp://localhost:1883"
```
Note that you can specify the deadzone and the broker location on the command line but the values shown are the defaults.

### MacOS

Untested


### Windows

Untested 


## Topics

Messages are sent to topics on the broker:

```
"xb/1/joysticks"
"xb/1/triggers"
"xb/1/buttons"
```

## Messages

### Buttons

Button presses generate discrete events as defined in the [xbevents](https://github.com/gherlein/xbevents) module.  Examples:

```
PADD_DOWN
PADD_UP
GUIDE_DOWN
GUIDE_UP
Y_DOWN
Y_UP
```

### Triggers

Triggers pulls generate discrete events as defined in the [xbevents](https://github.com/gherlein/xbevents) module.  Examples:

```
LT|0
LT|53
LT|107
LT|255
LT|80
LT|0
RT|78
RT|152
RT|147
RT|85
RT|0
```

### Joysticks

Joysticks generate discrete events as defined in the [xbevents](https://github.com/gherlein/xbevents) module.  Examples:

```
L|Y|3584|0
L|Y|21248|7424
L|Y|32767|19712
L|Y|32767|28160
L|Y|32767|32767
L|Y|32767|13312
L|Y|11776|0
L|X|-1536|0
```

## License

This project is released under the MIT License.  Please see details 
[here] (https://gherlein.mit-license.org).



