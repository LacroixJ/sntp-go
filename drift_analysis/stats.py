#!/usr/bin/env python3

import fileinput
import re
import datetime
import matplotlib.pyplot as plt



rtts = []

drifts = []

average_drifts_per_iter = []

prev_drift = 0



count = 0
fails = 0

for line in fileinput.input():
    count += 1

    if re.search(r'timeout', line):
        fails += 1
        continue


    tokens = line.split(', ')


    orig_clock = tokens[0]

    rtt = tokens[1]

    if rtt == '0s':
        continue
    else:
        rtt = float(rtt[:-2])
        rtts.append(rtt)


    drift = float(tokens[3][:-2])

    drift_delta = prev_drift - drift

    prev_drift = drift

    drifts.append(drift_delta)


    avg_drift_iter = ((sum(drifts) / len(drifts)) * 1000) / 10

    average_drifts_per_iter.append(avg_drift_iter)



avg_rtt = sum(rtts) / len(rtts)
print(f'Avg rtt: {avg_rtt:.2f}ms')

packet_loss = (fails/count) * 100
print(f'Packet loss: {packet_loss:.2f}%')


avg_drift = ((sum(drifts) / len(drifts)) * 1000) / 10
print(f'Average drift rate: {avg_drift} microseconds per second')


remove_vals = []
for x in average_drifts_per_iter:
    if x > 15:
        remove_vals.append(x)


for x in remove_vals:
    average_drifts_per_iter.remove(x)


print(max(average_drifts_per_iter[50:]))
plt.hist(average_drifts_per_iter[50:], bins=15)

plt.xlabel("microseconds")
plt.title("Average drift from previous iteration in microseconds")
plt.ylabel("count")
plt.show()
