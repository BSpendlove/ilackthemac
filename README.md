# ilackthemac
Golang project that scrapes Standards OUI and will return Organization and alternate name.

There are many projects out there on the internet, some open-source, others paid per requests... I want an excuse to learn more Go so I thought I'd build a simple API to do a look from a file and also create a pipeline which triggers automatically every hour to update the existing [standards.oui.ieee.org](https://standards-oui.ieee.org/oui/oui.txt) to an easier format to ingest.

If you want to run this locally and keep it up to date, then simply pull the repo every hour.