package controller

var prometheusSvc = `
kind: Service
apiVersion: v1
metadata: 
  name: prometheus
  namespace: prometheus
  labels: 
    app: prometheus
    
spec: 
  ports: 
    - name: http 
      port: 9090
      protocol: TCP
      targetPort: 9090
  selector: 
    k8s-app: prometheus
`
var prometheusStatefulSet = `apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: prometheus 
  namespace: kube-system
  labels:
    k8s-app: prometheus
    kubernetes.io/cluster-service: "true"
    addonmanager.kubernetes.io/mode: Reconcile
    version: v2.2.1
spec:
  serviceName: "prometheus"
  replicas: 1
  podManagementPolicy: "Parallel"
  updateStrategy:
   type: "RollingUpdate"
  selector:
    matchLabels:
      k8s-app: prometheus
  template:
    metadata:
      labels:
        k8s-app: prometheus
      annotations:
        scheduler.alpha.kubernetes.io/critical-pod: ''
    spec:
      priorityClassName: system-cluster-critical
      serviceAccountName: prometheus
      initContainers:
      - name: "init-chown-data"
        image: "busybox:latest"
        imagePullPolicy: "IfNotPresent"
        command: ["chown", "-R", "65534:65534", "/data"]
        volumeMounts:
        - name: prometheus-data
          mountPath: /data
          subPath: ""
      containers:
        - name: prometheus-server-configmap-reload
          image: "jimmidyson/configmap-reload:v0.1"
          imagePullPolicy: "IfNotPresent"
          args:
            - --volume-dir=/etc/config
            - --webhook-url=http://localhost:9090/-/reload
          volumeMounts:
            - name: config-volume
              mountPath: /etc/config
              readOnly: true
          resources:
            limits:
              cpu: 10m
              memory: 10Mi
            requests:
              cpu: 10m
              memory: 10Mi

        - name: prometheus-server
          image: "prom/prometheus:v2.2.1"
          imagePullPolicy: "IfNotPresent"
          args:
            - --config.file=/etc/config/prometheus.yml
            - --storage.tsdb.path=/data
            - --web.console.libraries=/etc/prometheus/console_libraries
            - --web.console.templates=/etc/prometheus/consoles
            - --web.enable-lifecycle
          ports:
            - containerPort: 9090
          readinessProbe:
            httpGet:
              path: /-/ready
              port: 9090
            initialDelaySeconds: 30
            timeoutSeconds: 30
          livenessProbe:
            httpGet:
              path: /-/healthy
              port: 9090
            initialDelaySeconds: 30
            timeoutSeconds: 30
          # based on 10 running nodes with 30 pods each
          resources:
            limits:
              cpu: 200m
              memory: 1000Mi
            requests:
              cpu: 200m
              memory: 1000Mi
            
          volumeMounts:
            - name: config-volume
              mountPath: /etc/config
            - name: prometheus-data
              mountPath: /data
              subPath: ""
      terminationGracePeriodSeconds: 300
      volumes:
        - name: config-volume
          configMap:
            name: prometheus-config
  volumeClaimTemplates:
  - metadata:
      name: prometheus-data
    spec:
      storageClassName: standard
      accessModes:
        - ReadWriteOnce
      resources:
        requests:
          storage: "16Gi"`
