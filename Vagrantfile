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
        :ip => '10.10.10.2'
  end

  config.vm.define :pxeclient1, autostart: false do |pxeclient1|

    pxeclient1.vm.provider :libvirt do |pxeclient1_vm|
      pxeclient1_vm.memory = 2048
      pxeclient1_vm.cpus = 2
      pxeclient1_vm.graphics_port = 5901
      pxeclient1_vm.graphics_ip = '0.0.0.0'
      pxeclient1_vm.storage :file, :size => '20G', :type => 'qcow2'
      pxeclient1_vm.boot 'network'
      pxeclient1_vm.boot 'hd'
    end

    pxeclient1.vm.network :private_network,
        :mac => '52:54:00:ff:00:01',
        :ip => '10.10.10.101'
  end
end

