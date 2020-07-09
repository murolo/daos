#!/bin/bash

post_provision_config_nodes() {
    # TODO: port this to Zypper
    #       or do we even need it any more?
    #if $CONFIG_POWER_ONLY; then
    #    rm -f /etc/yum.repos.d/*.hpdd.intel.com_job_daos-stack_job_*_job_*.repo
    #    yum -y erase fio fuse ior-hpc mpich-autoload               \
    #                 ompi argobots cart daos daos-client dpdk      \
    #                 fuse-libs libisa-l libpmemobj mercury mpich   \
    #                 openpa pmix protobuf-c spdk libfabric libpmem \
    #                 libpmemblk munge-libs munge slurm             \
    #                 slurm-example-configs slurmctld slurm-slurmmd
    #fi

    # remove to avoid conflicts
    if ! zypper --non-interactive rm python2-Fabric Modules && \
       [ ${PIPESTATUS[0]} -ne 104 ]; then
        echo "Failed to remove packages"
        exit 1
    fi
    zypper --non-interactive in avocado patch python2-Jinja2 pciutils lua-lmod

    if [ -n "$DAOS_STACK_GROUP_REPO" ]; then
         # rm -f /etc/yum.repos.d/*"$DAOS_STACK_GROUP_REPO"
        zypper --non-interactive ar "$REPOSITORY_URL"/"$DAOS_STACK_GROUP_REPO" daos-stack-group-repo
        zypper --non-interactive mr --gpgcheck-allow-unsigned-repo daos-stack-group-repo
        zypper --non-interactive --gpg-auto-import-keys --no-gpg-checks ref daos-stack-group-repo
    fi
    
    if [ -n "$DAOS_STACK_LOCAL_REPO" ]; then
        zypper --non-interactive ar --gpgcheck-allow-unsigned "$REPOSITORY_URL"/"$DAOS_STACK_LOCAL_REPO" daos-stack-local-repo
        zypper --non-interactive mr --no-gpgcheck daos-stack-local-repo
    fi
    
    if [ -n "$INST_REPOS" ]; then
        for repo in $INST_REPOS; do
            branch="master"
            build_number="lastSuccessfulBuild"
            if [[ $repo = *@* ]]; then
                branch="${repo#*@}"
                repo="${repo%@*}"
                if [[ $branch = *:* ]]; then
                    build_number="${branch#*:}"
                    branch="${branch%:*}"
                fi
            fi
            zypper --non-interactive ar --gpgcheck-allow-unsigned "${JENKINS_URL}"job/daos-stack/job/"${repo}"/job/"${branch//\//%252F}"/"${build_number}"/artifact/artifacts/leap15/ "$repo"
        done
    fi
    # need 15.1 repos for transition to 15.2
    zypper --non-interactive ar http://download.opensuse.org/distribution/leap/15.1/repo/oss/ leap15.1-oss
    # need to hack in 15.2 science:HPC repo as a test
    zypper --non-interactive ar https://download.opensuse.org/repositories/science:/HPC/openSUSE_Leap_15.2/ science-hpc
    zypper --non-interactive ref leap15.1-oss science-hpc
    zypper --non-interactive lr
    # need to remove ipmctl since 15.2 has 2.0 and 15.1 only had 1.0
    if ! zypper --non-interactive rm ipmctl && \
       [ ${PIPESTATUS[0]} -ne 104 ]; then
        echo "Failed to remove packages"
        exit 1
    fi

    #if [ -n "$INST_RPMS" ]; then
        #yum -y erase $INST_RPMS
    #fi
    if ! zypper --non-interactive in ed nfs-client sudo nfs-kernel-server \
                                     $INST_RPMS; then
        rc=${PIPESTATUS[0]}
        for file in /etc/zypp/repos.d/*.repo; do
            echo "---- $file ----"
            cat "$file"
        done
        exit "$rc"
    fi
}
