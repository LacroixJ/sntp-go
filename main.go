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
const ntp_offset = 2208988800 //70 years in seconds (diff between 1900 and 1970 unix time)

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
    Receive_timestamp uint32
    Receive_timestamp_fractional uint32
    Transmit_timestamp uint32
    Transmit_timestamp_fractional uint32
}

func main(){

    number_of_points := 4320 //12 hours in 10 second increments is 4320
    fmt.Println("Time, RTT, Omega, Smoothed Omega, Corrected Time")

    var rolling_omega []time.Duration

    var smoothed_omega time.Duration

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
        corrected_time := current_time.Add(smoothed_omega)
        fmt.Printf("%v, %v, %v, %v, %v\n", current_time, result[0],
                    result[1], smoothed_omega, corrected_time)

        time.Sleep(time.Second * 10)
    }

}

func average(slice []time.Duration) time.Duration{

    var sum time.Duration
    for _, x := range slice{
       sum += x
    }

    length := len(slice)

    return time.Duration(int64(sum) / int64(length))

}


func calc_ping_offset() [2]time.Duration {


    t := get_timestamps()

    //fmt.Println("Timestamps are: ", t)

    pingtime := t[2].Sub(t[3]) + t[0].Sub(t[1])
    //fmt.Println("Ping is this many millisseconds", pingtime)

    es_offset := (t[2].Sub(t[3]) - (t[0].Sub(t[1]))) / 2.0

    offset_seconds := es_offset

    answers := [2]time.Duration{pingtime, offset_seconds}
    return answers

}

func get_timestamps() [4]time.Time {
    var times [4]time.Time

    t3 := time.Now().UTC() //send timestamp

    times[3] = t3

    //going to assume t2=t1

    t2, t1 := send_sntp_packet(ntp_server) //recieve and transmit timestamps

    empty := time.Time{}
    if t2 == empty{
        return [4]time.Time{empty, empty, empty, empty}
    }

    times[2] = t2
    times[1] = t1

    t0 := time.Now().UTC() //local recieve timestamp

    times[0] = t0

    return times
}



func send_sntp_packet(server string) (time.Time, time.Time) {
    conn, err := net.Dial("udp", server)
    if err != nil{
        fmt.Println("Error dialing server!,", err)
        return time.Time{}, time.Time{}
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
        return time.Time{}, time.Time{}
    }

    myntp_packet := ntp_packet{}

    //convert the buffer to an ntp_packet struct
    buf := bytes.NewReader(buffer)
    err = binary.Read(buf, binary.BigEndian, &myntp_packet)
    if err != nil {
        fmt.Println("Converting back to struct failed:", err)
        return time.Time{}, time.Time{}
    }

    //fmt.Println("returned time is ", myntp_packet.Transmit_timestamp - ntp_offset)

    seconds_r := myntp_packet.Receive_timestamp
    fraction_r := myntp_packet.Receive_timestamp_fractional

    recieve := ntp_time_to_unix(seconds_r, fraction_r)


    seconds_t := myntp_packet.Transmit_timestamp
    fraction_t := myntp_packet.Transmit_timestamp_fractional

    transmit := ntp_time_to_unix(seconds_t, fraction_t)

    return recieve, transmit
}

func create_client_ntp_packet() ntp_packet {
    client_packet := ntp_packet{}

    //LI = 0, VN=4 MODE = 3 (client) total 100011
    client_packet.Flags = 35

    return client_packet
}


func ntp_time_to_unix(ntptime uint32, ntptimefrac uint32) time.Time{
    seconds := float64(ntptime) - ntp_offset

    //thanks to this page for showing me how to easily convert
    //https://medium.com/learning-the-go-programming-language/
    //lets-make-an-ntp-client-in-go-287c4b9a969f

    nanos := (int64(ntptimefrac) * 1e9) >> 32

    return time.Unix(int64(seconds), nanos)
}
