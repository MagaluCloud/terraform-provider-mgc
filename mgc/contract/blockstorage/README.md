# Block Storage contracts

The local API accepts attachment and fails the following read. Terraform must
retain the association, plan explicit replacement, detach it before reattaching,
and converge to an empty plan. The fixture rejects volume deletion and duplicate
attachments. Unit tests also exercise cancellation immediately after acceptance
and ensure a rejected attach does not create state. Production polling keeps its
10-second settling interval. These contracts do not validate real volume health.
