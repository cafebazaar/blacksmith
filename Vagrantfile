# -*- mode: ruby -*-
# # vi: set ft=ruby :

unless Vagrant.has_plugin?("vagrant-libvirt")
  raise Vagrant::Errors::VagrantError.new, "Please install the vagrant-libvirt plugin running 'vagrant plugin install vagrant-libvirt'"
end

Vagrant.configure("2") do |config|

  config.vm.define :pxeserver do |pxeserver|
    pxeserver.vm.box = "naelyn/ubuntu-trusty64-libvirt"

    pxeserver.vm.provider :libvirt do |pxeserver_vm|
      pxeserver_vm.memory = 2048
      pxeserver_vm.cpus = 2
      pxeserver_vm.graphics_port = 5900
      pxeserver_vm.graphics_ip = '0.0.0.0'
    end

    pxeserver.vm.provision :shell, path: 'vagrant_provision.sh'

    pxeserver.vm.network :private_network,
        :ip => '10.10.10.2',
        :libvirt__dhcp_enabled => 'false'
  end

  config.vm.define :pxeserver2, autostart: false do |pxeserver2|
    pxeserver2.vm.box = "naelyn/ubuntu-trusty64-libvirt"

    pxeserver2.vm.provider :libvirt do |pxeserver2_vm|
      pxeserver2_vm.memory = 2048
      pxeserver2_vm.cpus = 2
      pxeserver2_vm.graphics_port = 5901
      pxeserver2_vm.graphics_ip = '0.0.0.0'
    end

    pxeserver2.vm.provision :shell, path: 'vagrant_provision.sh'

    pxeserver2.vm.network :private_network,
        :ip => '10.10.10.3',
        :libvirt__dhcp_enabled => 'false'
  end

  config.vm.define :pxeclient1, autostart: false do |pxeclient1|

    pxeclient1.vm.network :private_network,
        :mac => '52:54:00:ff:00:01',
        :ip => '10.10.10.100',                # Dummy
        :libvirt__dhcp_enabled => 'false'

    pxeclient1.vm.provider :libvirt do |pxeclient1_vm|
      pxeclient1_vm.memory = 2048
      pxeclient1_vm.cpus = 2
      pxeclient1_vm.graphics_port = 5910
      pxeclient1_vm.graphics_ip = '0.0.0.0'
      pxeclient1_vm.storage :file, :size => '20G', :type => 'qcow2'
      pxeclient1_vm.boot 'network'
    end
  end

  config.vm.define :pxeclient2, autostart: false do |pxeclient2|

    pxeclient2.vm.network :private_network,
        :mac => '52:54:00:ff:00:02',
        :ip => '10.10.10.101',                # Dummy
        :libvirt__dhcp_enabled => 'false'

    pxeclient2.vm.provider :libvirt do |pxeclient2_vm|
      pxeclient2_vm.memory = 2048
      pxeclient2_vm.cpus = 2
      pxeclient2_vm.graphics_port = 5911
      pxeclient2_vm.graphics_ip = '0.0.0.0'
      pxeclient2_vm.storage :file, :size => '20G', :type => 'qcow2'
      pxeclient2_vm.boot 'network'
    end
  end
end
