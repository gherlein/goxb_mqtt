NAME   = goxb_mqtt
FILE   = ./${NAME}

build: dependencies 
	go build 

${FILE}: build

dependencies:
	go get ./...

install: ${FILE}
	sudo systemctl stop ${NAME}
	sudo cp ${FILE} /usr/local/bin/
	sudo cp ${FILE}.toml /etc/
	sudo cp ${FILE}.service /etc/systemd/system
	sudo systemctl daemon-reload
	sudo systemctl enable ${NAME}
	sudo systemctl start ${NAME}

stop:
	sudo service ${NAME} stop

start:
	sudo service ${NAME} start

restart:
	sudo service ${NAME} restart

clean:
	-rm -f ${FILE}
	-rm -f *~

run:
	sudo ./goxb_mqtt --deadzone=512 --broker="tcp://localhost:1883"



