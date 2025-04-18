TARGET = vidify

build:
	go build -o $(TARGET).exe .
run:
	go run .

clean:
	rm -f $(TARGET)

.PHONY: build run clean
