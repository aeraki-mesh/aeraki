#!/bin/bash


#set -x

function LabelIstioInjectLabel() 
{  
    ns=$1
	echo $ns
	label=`kubectl get po -n istio-system  |grep istiod | awk '{print $1}'  |xargs kubectl get po  -o yaml  -n istio-system  |grep -A 1 REVIS |grep value:  |awk  '{print $2}'`
	echo $label
	if [ $label != "" ];then
		kubectl label namespace $ns istio.io/rev=$label --overwrite
	else 
		kubectl label namespace $ns istio-injection=enabled --overwrite=true
	fi
    return 0;  
}

