def helm_resource(name, chart, namespace='default', flags=[], deps=[], resource_deps=[], labels=[], pod_readiness='wait'):
    command = ['mise', 'exec', '--', 'go', 'run', '../tools/helm-resource']
    k8s_custom_deploy(
        name,
        apply_cmd=command + ['apply', name, namespace, chart] + flags,
        delete_cmd=command + ['delete', name, namespace],
        deps=deps + ['../tools/helm-resource'],
    )
    k8s_resource(
        name,
        resource_deps=resource_deps,
        labels=labels,
        pod_readiness=pod_readiness,
    )
