package fleetdb

// TODO: move these consts into the hollow-toolbox to share between controllers.

const (
	// fleetdb BMC address attribute key
	bmcIPAddressAttributeKey = "address"

	// fleetdb namespace prefix the data is stored in.
	fleetdbNSPrefix = "sh.hollow.bioscfg"

	// server vendor, model attributes are stored in this namespace.
	serverVendorAttributeNS = fleetdbNSPrefix + ".server_vendor_attributes"

	// server service server serial attribute key
	serverSerialAttributeKey = "serial"

	// server service server model attribute key
	serverModelAttributeKey = "model"

	// server service server vendor attribute key
	serverVendorAttributeKey = "vendor"
)
