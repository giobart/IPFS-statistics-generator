
env GOOS=linux GOARCH=arm GOARM=6 go build -o bin/ipfs-statistics-generator
scp bin/ipfs-statistics-generator pi@raspberrypigio2:/home/pi/tools
