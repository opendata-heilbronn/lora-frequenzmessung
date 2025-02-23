# LORA Frequency Measurement

A "simple" way to measure crowd density using Bluetooth and LoRa technology.

## Introduction

"LOAR Frequency Measurement" is a project that aims to measure crowd density in a given area. The project uses Bluetooth technology to detect the presence of people and LoRa technology to send the data to a server. The server then processes the data and provides a real-time visualization of the crowd density in the area.


# Install 
ENV vars ? well todo 

# migration 
install
```
mac
brew install golang-migrate
```
Create migration
```
migrate create -ext sql -dir db/migrations -seq create_users_table
```
Run migration
```
migrate -database "postgres://timescaledb:password@localhost:5432/postgres?sslmode=disable"  -path db/migrations up
```