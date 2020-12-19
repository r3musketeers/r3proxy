#!/usr/bin/env python3
import sys

from pandas import read_csv
from matplotlib import pyplot

series = read_csv(
  sys.argv[1],
  sep=' ',
  squeeze=True,
)

series['total_threads'] = series['client_nodes'] * series['threads_per_client']

print(series)

series.plot(x='avg_throughput', y='latency_90th')
pyplot.show()
