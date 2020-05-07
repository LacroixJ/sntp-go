package main

import (
	"bytes"
	"fmt"
	"net"
	"time"
	"bufio"
	"encoding/binary"
        "unsafe"
)

var ntp_server string = "66.70.179.176:123" //one of 0.pool.ntp.org addresses
var ntp_offset uint32 = 2208988800 //70 years in seconds (diff between 1900 and 1970 unix time)

type ntp_packet struct { //total size is 48 bytes
    Flags byte //LI, VN, Mode
    Peer_clock_stratum byte
    Peer_polling_interval byte
    Peer_clock_precision byte
    Root_delay uint32
    Root_dispersion uint32
    Reference_id uint32
    Reference_time uint64
    Origin_timestamp uint64
    Receive_timestamp uint64
    Transmit_timestamp uint32 //this is the only number I'm going to care about
    Transmit_timestamp_fractional uint32
}

func main(){

    number_of_points := 4320 //12 hours in 10 second increments is 4320
    fmt.Println("Time, RTT, Omega, Smoothed Omega, Corrected Time")

    var rolling_omega []float64

    var smoothed_omega float64

    for i := 1; i < number_of_points; i++{
        result := calc_ping_offset()

        if result[0] != 0 && result[1] != 0{ //succesfull response
            rolling_omega = append(rolling_omega, result[1])
            if len(rolling_omega) > 8{
                rolling_omega = rolling_omega[1:]
            }
        }

        current_time := time.Now().UTC()
        smoothed_omega = average(rolling_omega)
        smoothed_converted := time.Duration(smoothed_omega * 1000000) * time.Microsecond
        corrected_time := current_time.Add(smoothed_converted)
        fmt.Printf("%v, %v, %v, %v, %v\n", current_time, result[0], result[1],
                    smoothed_omega, corrected_time)

        time.Sleep(time.Second * 10)
    }

}

func average(slice []float64) float64{

    var sum float64
    for _, x := range slice{
       sum += x
    }

    return sum/ float64(len(slice))

}


func calc_ping_offset() [2]float64 {


    t := get_timestamps()

    //fmt.Println("Timestamps are: ", t)

    pingtime := float64((t[2] - t[3]) + (t[0] - t[1])) / 1000000000
    //fmt.Println("Ping is this many millisseconds", pingtime)

    es_offset := ((t[2] - t[3]) - (t[0]-t[1])) / 2.0

    offset_seconds := float64(es_offset) / 1000000000

    answers := [2]float64{pingtime, offset_seconds}
    return answers

}

func get_timestamps() [4]int64 {
    t3 := time.Now().UTC().UnixNano()

    var times [4]int64

    times[3] = t3

    //going to assume t2=t1

    t2 := int64(send_sntp_packet(ntp_server))

    if t2 == 0{
        return [4]int64{0, 0, 0, 0}
    }

    times[2] = t2 * 1000000000
    times[1] = times[2]

    t0 := time.Now().UTC().UnixNano()

    times[0] = t0

    return times
}



func send_sntp_packet(server string) uint32 {
    conn, err := net.Dial("udp", server)
    if err != nil{
        fmt.Println("Error dialing server!,", err)
        return 0
    }

    //structs need to be encoded into a byte array before being sent
    binary.Write(conn, binary.BigEndian, create_client_ntp_packet())

    //set a read timeout
    conn.SetReadDeadline(time.Now().Add(1000 * time.Millisecond))

    //read the data into a buffer
    buffer := make([]byte, unsafe.Sizeof(ntp_packet{}))
    _, err = bufio.NewReader(conn).Read(buffer)
    if err != nil {
        fmt.Println(err)
        return 0
    }

    myntp_packet := ntp_packet{}

    //convert the buffer to an ntp_packet struct
    buf := bytes.NewReader(buffer)
    err = binary.Read(buf, binary.BigEndian, &myntp_packet)
    if err != nil {
        fmt.Println("Converting back to struct failed:", err)
        return 0
    }

    //fmt.Println("returned time is ", myntp_packet.Transmit_timestamp - ntp_offset)


    return myntp_packet.Transmit_timestamp - ntp_offset
}

func create_client_ntp_packet() ntp_packet {
    client_packet := ntp_packet{}

    //LI = 0, VN=4 MODE = 3 (client) total 100011
    client_packet.Flags = 35

    return client_packet
}
