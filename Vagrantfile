# -*- mode: ruby -*-
# vi: set ft=ruby :

Vagrant.configure("2") do |config|
  config.vm.box = "ubuntu/focal64"
  
  # Имя виртуальной машины
  config.vm.hostname = "k8s-dev"
  
  # Настройки VirtualBox
  config.vm.provider "virtualbox" do |vb|
    vb.name = "k8s-go-service"
    vb.memory = "10240"
    vb.cpus = 4
    
    # Включаем вложенную виртуализацию для Minikube
    vb.customize ["modifyvm", :id, "--nested-hw-virt", "on"]
  end
  
  # Проброс портов для доступа с host machine
  # Grafana
  config.vm.network "forwarded_port", guest: 3000, host: 3000, host_ip: "127.0.0.1"
  
  # Prometheus
  config.vm.network "forwarded_port", guest: 9090, host: 9090, host_ip: "127.0.0.1"
  
  # Go Service (через Ingress)
  config.vm.network "forwarded_port", guest: 80, host: 8080, host_ip: "127.0.0.1"
  
  # Kubernetes Dashboard (опционально)
  config.vm.network "forwarded_port", guest: 8001, host: 8001, host_ip: "127.0.0.1"
  
  # Redis (для отладки)
  config.vm.network "forwarded_port", guest: 6379, host: 6379, host_ip: "127.0.0.1"
  
  # Locust (для нагрузочного тестирования)
  config.vm.network "forwarded_port", guest: 8089, host: 8089, host_ip: "127.0.0.1"
  
  # Синхронизация папки проекта
  config.vm.synced_folder ".", "/vagrant"
  
  # Provisioning скрипт для установки необходимых компонентов
  config.vm.provision "shell", inline: <<-SHELL
    set -e
    
    echo "=== Обновление системы ==="
    apt-get update
    apt-get upgrade -y
    
    echo "=== Установка базовых пакетов ==="
    apt-get install -y \
      curl \
      wget \
      git \
      vim \
      htop \
      net-tools \
      apt-transport-https \
      ca-certificates \
      software-properties-common \
      gnupg \
      lsb-release
    
    echo "=== Установка Docker ==="
    # Удаляем старые версии
    apt-get remove -y docker docker-engine docker.io containerd runc || true
    
    # Добавляем репозиторий Docker
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg
    echo "deb [arch=amd64 signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null
    
    apt-get update
    apt-get install -y docker-ce docker-ce-cli containerd.io
    
    # Добавляем пользователя vagrant в группу docker
    usermod -aG docker vagrant
    
    # Запускаем Docker
    systemctl enable docker
    systemctl start docker
    
    echo "=== Установка Go 1.22 ==="
    GO_VERSION="1.22.0"
    wget -q https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz
    rm -rf /usr/local/go
    tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz
    rm go${GO_VERSION}.linux-amd64.tar.gz
    
    # Настройка PATH для всех пользователей
    echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
    echo 'export GOPATH=$HOME/go' >> /etc/profile
    echo 'export PATH=$PATH:$GOPATH/bin' >> /etc/profile
    
    # Для пользователя vagrant
    echo 'export PATH=$PATH:/usr/local/go/bin' >> /home/vagrant/.bashrc
    echo 'export GOPATH=$HOME/go' >> /home/vagrant/.bashrc
    echo 'export PATH=$PATH:$GOPATH/bin' >> /home/vagrant/.bashrc
    
    echo "=== Установка kubectl ==="
    curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
    install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl
    rm kubectl
    
    echo "=== Установка Minikube ==="
    curl -LO https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64
    install minikube-linux-amd64 /usr/local/bin/minikube
    rm minikube-linux-amd64
    
    echo "=== Установка Helm ==="
    curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
    
    echo "=== Установка Python и Locust ==="
    apt-get install -y python3 python3-pip
    pip3 install locust
    
    echo "=== Настройка автодополнения kubectl ==="
    echo 'source <(kubectl completion bash)' >> /home/vagrant/.bashrc
    echo 'alias k=kubectl' >> /home/vagrant/.bashrc
    echo 'complete -F __start_kubectl k' >> /home/vagrant/.bashrc
    
    echo "=== Настройка завершена ==="
    echo "Docker version: $(docker --version)"
    echo "Go version: $(su - vagrant -c '/usr/local/go/bin/go version')"
    echo "kubectl version: $(kubectl version --client --short)"
    echo "Minikube version: $(minikube version --short)"
    echo "Helm version: $(helm version --short)"
    echo "Locust version: $(locust --version)"
    
  SHELL
end
