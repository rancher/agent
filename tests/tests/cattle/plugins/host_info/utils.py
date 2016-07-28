import re


def semver_trunk(version, vrm_vals=3):
        '''
        vrm_vals: is a number representing the number of
        digits to return. ex: 1.8.3
          vmr_val = 1; return val 1
          vmr_val = 2; return val 1.8
          vmr_val = 3; return val 1.8.3
        '''
        if version:
            return {
                1: re.search('(\d+)', version).group(),
                2: re.search('(\d+\.)?(\d+)', version).group(),
                3: re.search('(\d+\.)?(\d+\.)?(\d+)', version).group(),
            }.get(vrm_vals, version)

        return version
