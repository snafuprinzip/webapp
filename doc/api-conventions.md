API resources should use the traditional REST pattern:

- GET /\<resourceNamePlural\>
  - Retrieve a list of type \<resourceName\>, e.g. GET /pods returns a list of Pods.
- POST /\<resourceNamePlural\>
  - Create a new resource from the JSON object provided by the client.
- GET /\<resourceNamePlural\>/\<name\>      
  - Retrieves a single resource with the given name, e.g. GET /pods/first returns a Pod named 'first'. Should be constant time, and the resource should be bounded in size.
- DELETE /\<resourceNamePlural\>/\<name\>   
  - Delete the single resource with the given name. DeleteOptions may specify gracePeriodSeconds, the optional duration in seconds before the object should be deleted. Individual kinds may declare fields which provide a default grace period, and different kinds may have differing kind-wide default grace periods. A user provided grace period overrides a default grace period, including the zero grace period ("now").
- DELETE /\<resourceNamePlural\>       
  - Deletes a list of type \<resourceName\>, e.g. DELETE /pods a list of Pods.
- PUT /\<resourceNamePlural\>/\<name\>      
  - Update or create the resource with the given name with the JSON object provided by the client.
- PATCH /\<resourceNamePlural\>/\<name\>
  - Selectively modify the specified fields of the resource. See more information below.
- GET /\<resourceNamePlural\>?watch=true
  - Receive a stream of JSON objects corresponding to changes made to any resource of the given kind over time.
