# -*- mode: ruby -*-
# # vi: set ft=ruby :

unless Vagrant.has_plugin?("vagrant-libvirt")
  raise Vagrant::Errors::VagrantError.new, "Please install the vagrant-libvirt plugin running 'vagrant plugin install vagrant-libvirt'"
end

num_servers = (ENV['NUM_SERVERS'] || 3).to_i
num_clients = (ENV['NUM_CLIENTS'] || 3).to_i
workspace_path = (ENV['WORKSPACE'] || "/vagrant/workspaces/current")
overwrite_dockers = (ENV['OVERWRITE_DOCKERS'] || "1")

Vagrant.configure("2") do |config|

  num_servers.times do |n|
    config.vm.define "pxeserver#{n+1}" do |pxeserver|
      pxeserver.vm.box = "naelyn/ubuntu-trusty64-libvirt"

      pxeserver.vm.provider :libvirt do |pxeserver_vm|
        pxeserver_vm.memory = 2048
        pxeserver_vm.cpus = 2
      end

      pxeserver.vm.provision :shell, inline: "/vagrant/vagrant_provision.sh #{num_servers} #{n+1} #{workspace_path} #{overwrite_dockers}"

      pxeserver.vm.network :private_network,
          :libvirt__network_name => "pxenetwork",
          :libvirt__netmask => "255.255.255.0",
          :libvirt__dhcp_enabled => false,
          :mac => "52:54:00:ff:00:0#{n+1}",
          :ip => "10.10.10.1#{n+1}"
      pxeserver.vm.hostname = "pxeserver#{n+1}"
    end
  end

  num_clients.times do |n|
    config.vm.define "pxeclient#{n+1}", autostart: false do |pxeclient|

      pxeclient.vm.network :private_network,
          :libvirt__network_name => "pxenetwork",
          :libvirt__netmask => "255.255.255.0",
          :libvirt__dhcp_enabled => false,
          :mac => "52:54:00:ff:00:1#{n+1}",
          :ip => "10.10.10.10#{n+1}"

      pxeclient.vm.provider :libvirt do |pxeclient_vm|
        pxeclient_vm.memory = 2048
        pxeclient_vm.cpus = 2
        pxeclient_vm.storage :file, :size => '20G', :type => 'qcow2'
        pxeclient_vm.boot 'network'
      end
	end
  end

end
