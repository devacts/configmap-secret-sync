The manager calls Reconcile for you. The flow is:
manager starts
└── starts internal informer/cache for types you registered in SetupWithManager
    └── informer gets Add/Update/Delete event for a ConfigMap
        └── manager enqueues the object's namespace/name
            └── manager's internal worker dequeues it
                └── calls your Reconcile(ctx, req)

