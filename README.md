# switch-traffic-to-mackerel

## usage

```
export MACKEREL_API_KEY=xxxx
./switch-traffic-to-mackerel -target 192.0.2.1 -mibs ifHCInOctets,ifHCOutOctets -include-interface 'ge-0/0/\d+$|ae0$' -skip-down-link-state -name sw1
```

