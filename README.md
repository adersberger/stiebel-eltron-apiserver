# Stiebel Eltron ISG API Server
![Stiebel Eltron ISG API Server](logo.png)
Exposes the Stiebel Eltron ISG statistics and configuration values as REST API. 
Could be used to integrate Stiebel Eltron devices into home automation software like Loxone or OpenHAB or to control the heatpump with your own AI magic.

## Usage guide
 * Download most current release binaries - e.g. on a Raspberry Pi: `wget https://github.com/adersberger/stiebel-eltron-apiserver/releases/download/0.1/isg-apiserver-linux-arm` and make the downloaded binary executable `chmod +x isg-apiserver-linux-arm`
 * Start the server and specify ISG IP address as first command line argument: `isg-apiserver-<TARGET-PLATFORM> <ISG IP address>`
 
## Available endpoints
 
**http://localhost:5432/stats**
Returns a JSON formatted document with all ISG-exposed statistics and configuration values.
 
**http://localhost:5432/value/val16?new=24.0**
Sets a new value for a configuration value. The following configuration values are available:

| Config value  | Path identifier |
| ------------- | ------------- |
| Temperatur Innenraum / Heizen Komforttemperatur  | val16  |
| Temperatur Warmwasser  | val22  |
| Heizen Eco-Temperatur  | val17  |
| Steigung Heizkurve  | val25  |
| Warmwasser Komforttemperatur  | val22  |
| Warmwasser Eco-Temperatur  | val23  |

## Building
To build the binaries all you have to do is to issue the following command: `make`

## Contributing
Is there anything missing? Do you have ideas for new features or improvements? You are highly welcome to contribute your improvements to the project. All you have to do is to fork this repository, improve the code and issue a pull request.

## Maintainer
Josef Adersberger (@adersberger)

## License
This software is provided under the Apache License, Version 2.0 license.
See the `LICENSE` file for details.