bootDir = File.expand_path(File.dirname(__FILE__) + '/.vagrant/bootstrap');

require 'yaml'
require bootDir + '/util.rb'

settings = YAML::load(File.read(bootDir + '/config.yaml'))

unless Vagrant.has_plugin?("vagrant-vbguest")
    raise 'Please, run "vagrant plugin install vagrant-vbguest".'
end

unless Vagrant.has_plugin?("vagrant-hostmanager")
    raise 'Please, run "vagrant plugin install vagrant-hostmanager".'
end


Vagrant.configure("2") do |config|

    config.hostmanager.enabled = true
    config.hostmanager.include_offline = true
    config.hostmanager.ignore_private_ip = false
    config.hostmanager.manage_guest = false
    config.hostmanager.manage_host = true
    config.hostmanager.aliases = settings["aliases"]

    config.vm.define settings["machine_name"] do |machine|

         machine.vm.box = settings["box"]
         machine.vm.box_version = settings["version"]
         machine.vm.network settings["net"], type: settings["net_type"]


         machine.vm.synced_folder ".", settings["sync_to"],
             type: "virtualbox"

         machine.vm.provider settings["provider"] do |vb|
             vb.cpus = settings["cpus"]
             vb.memory = settings["memory"]
             vb.name = settings["machine_name"]
             vb.customize ["modifyvm", :id, "--natdnsproxy1", "on"]
         end

         machine.hostmanager.ip_resolver = proc do |vm, resolving_vm|
             read_ip_address(vm)
         end

         provision(machine, bootDir, settings)

    end
end
