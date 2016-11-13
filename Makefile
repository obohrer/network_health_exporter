all: glide_install clean network_health_exporter test

binary := "network_health_exporter"

network_health_exporter:
	go build $(binary)

glide_install:
	glide install

run: clean network_health_exporter
	./$(binary) --config conf.json

test:
	go test network_health_exporter

clean:
	go clean
	if [ -a $(binary) ] ; \
  	then \
    	rm $(binary) ; \
	fi;
